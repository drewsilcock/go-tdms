// The stream reader allows iterative reading of values from a TDMS file for a
// particular channel.
//
// It uses batching to speed up reads, with functions that return either the
// batches as slices or the individual values. The stream reader that returns
// individual values still uses batching internally, it just helpfully unwraps
// the slice for you.

package tdms

import (
	"encoding/binary"
	"fmt"
	"io"
	"iter"
)

// BatchStreamReader returns an iterator that yields batches of values from the
// channel. Each batch is a slice of values read from the underlying file. Use
// the [BatchSize] option to control how many values are read in each batch.
//
// Important: The same underlying slice is reused for each batch to avoid
// allocations. If you need to retain batch data beyond the current iteration,
// you must copy it to your own buffer. For reading all data into a single
// slice, use the ReadData*All methods on [Channel] instead.
func BatchStreamReader[TRaw any](ch *Channel, options []ReadOption) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		opts := renderReadOptions(options)

		// For DAQmx data, we need the scaler for two purposes:
		//
		// 1. To determine the actual data type (when RawDataType == DataTypeDAQmxRawData)
		// 2. To get layout info (buffer index, byte offset) for interleaved reading
		//
		// Some files (e.g. multi-rate DAQmx) store the actual data type in the
		// raw data index rather than DataTypeDAQmxRawData, but still use DAQmx
		// scalers for buffer-based interleaving. We try to extract the scaler
		// from the channel's scaler chain first. If not found there (e.g. when
		// NI_Number_Of_Scales is 0), we fall back to the object's raw data
		// index scalers stored on the data chunk.
		var daqmxScaler *DAQmxScaler
		interpreterDataType := ch.RawDataType

		// Try the channel's scaler chain first.
		if opts.daqmxScaleIndex < len(ch.scaler.scalers) {
			if s, ok := ch.scaler.scalers[opts.daqmxScaleIndex].(*DAQmxScaler); ok {
				daqmxScaler = s
			}
		}

		// If not found in the scaler chain, try the first chunk's raw data
		// index scalers. These are always present for DAQmx objects, even
		// when the channel's scaler chain is empty.
		if daqmxScaler == nil && len(ch.dataChunks) > 0 && ch.dataChunks[0].daqMXScalers != nil {
			if s, ok := ch.dataChunks[0].daqMXScalers[opts.daqmxScaleIndex]; ok {
				daqmxScaler = &s
			} else if len(ch.dataChunks[0].daqMXScalers) == 1 {
				// Fall back to the only available scaler when the exact
				// scaleIndex doesn't match. This handles files where
				// scaleID is 0xFFFFFFFF (e.g. multi-rate DAQmx files
				// that store the actual data type instead of
				// DataTypeDAQmxRawData).
				for _, s := range ch.dataChunks[0].daqMXScalers {
					daqmxScaler = &s
					break
				}
			}
		}

		if ch.RawDataType == DataTypeDAQmxRawData {
			if daqmxScaler == nil {
				yield(nil, fmt.Errorf("invalid DAQmx scale index: %d", opts.daqmxScaleIndex))
				return
			}

			var err error
			interpreterDataType, err = daqmxScaler.OutputType(nil)
			if err != nil {
				yield(nil, fmt.Errorf("failed to get DAQmx scaler output type: %w", err))
				return
			}
		}

		interpret, err := getInterpreter(interpreterDataType)
		if err != nil {
			yield(nil, fmt.Errorf("unable to get byte interpreter: %w", err))
			return
		}

		if opts.batchSize == 0 {
			opts.batchSize = 2056
			if ch.RawDataType == DataTypeString {
				// Strings are generally much larger than individual ints or
				// floats, so we use much smaller default batch size.
				opts.batchSize = 256
			}
		}

		// If we have fewer data points in total than a single batch size, we
		// can allocate only what we need.
		batchSize := min(opts.batchSize, int(ch.totalNumValues))

		dataSize := ch.RawDataType.Size()
		if ch.RawDataType == DataTypeDAQmxRawData {
			dataSize = daqmxScaler.dataType.Size()
		}

		scaler, _ := NewMultiscaler(ch.RawDataType, nil)
		if opts.shouldScale {
			scaler = ch.scaler
			if err := scaler.Allocate(batchSize); err != nil {
				yield(nil, fmt.Errorf("failed to allocate scaler for channel %s: %w", ch.Name, err))
				return
			}
		}

		buf := make([]byte, batchSize*dataSize)
		bufLen := uint64(len(buf))
		batch := make([]TRaw, batchSize)
		r := ch.reader

		for _, chunk := range ch.dataChunks {
			if _, err := r.Seek(chunk.offset, io.SeekStart); err != nil {
				yield(nil, fmt.Errorf("failed to seek chunk %d: %w", chunk.offset, err))
				return
			}

			bytesRead := uint64(0)

			// Special case for strings, where the indices into the strings are
			// stored at the beginning of the chunk.
			strOffsets := []uint32{0}
			if ch.RawDataType == DataTypeString {
				strOffsetsBytes := make([]byte, chunk.numValues*4)
				n, err := r.Read(strOffsetsBytes)
				if err != nil {
					yield(nil, fmt.Errorf("failed to read chunk %d string offsets: %w", chunk.offset, err))
					return
				}
				bytesRead += uint64(n)

				for i := range chunk.numValues {
					strOffsets = append(strOffsets, chunk.order.Uint32(strOffsetsBytes[i*4:]))
				}
			}

			// For strings, we need to keep track of the current index that
			// we're processing so that we can get the offset for that value.
			valuesProcessed := 0

			for {
				// We don't want to read past the end of the chunk.
				bytesLeft := chunk.size - bytesRead
				if bytesLeft <= 0 {
					break
				}

				// For DAQmx/interleaved data, chunk.size may reflect the
				// total size across all scalers or the full buffer layout,
				// but we only read one scaler at a time. Stop when we've
				// consumed all values for this chunk.
				if uint64(valuesProcessed) >= chunk.numValues {
					break
				}

				// For strings, our buf starts with length 0 because data size
				// is 0. Now that we know how long each value is, we can make
				// buf big enough to hold the values for this batch.
				if ch.RawDataType == DataTypeString {
					numValuesLeft := 0
					for i := valuesProcessed; i < int(chunk.numValues); i++ {
						numValuesLeft++
					}

					requiredNumValues := min(batchSize, numValuesLeft)

					requiredBufLen := uint32(0)
					for i := valuesProcessed; i < valuesProcessed+requiredNumValues; i++ {
						requiredBufLen += strOffsets[i+1] - strOffsets[i]
					}

					bufLen = uint64(requiredBufLen)
					if cap(buf) < int(requiredBufLen) {
						buf = make([]byte, requiredBufLen)
					} else {
						buf = buf[:requiredBufLen]
					}
				}

				if bufLen > bytesLeft {
					// This retains capacity.
					buf = buf[:bytesLeft]
				} else {
					buf = buf[:bufLen]
				}

				n := 0
				var err error

				if !chunk.isInterleaved && !chunk.isDAQmx {
					n, err = io.ReadFull(r, buf)
				} else {
					// DAQmx is an interleaved format, just one with more advanced interleaving capabilities.

					// You aren't allowed to have interleaved variable-length
					// data channels.
					if dataSize == 0 {
						yield(
							nil,
							fmt.Errorf(
								"%w: interleaved data chunks cannot contains variable-length data types",
								ErrInvalidFileFormat,
							),
						)
						return
					}

					initialOffset := int64(0)
					stride := chunk.stride

					if chunk.isDAQmx {
						byteOffset := daqmxScaler.offsetWithinStride
						if daqmxScaler.scalerType == daqmxScalerTypeDigitalLine {
							// Digital line specifies offset in bits, not bytes.
							byteOffset = byteOffset / 8
						}

						stride = int64(chunk.daqMXBufferWidths[daqmxScaler.rawBufferIndex]) - int64(dataSize)
						for i := range daqmxScaler.rawBufferIndex {
							initialOffset += int64(chunk.daqMXBufferSizes[i])
						}
						initialOffset += int64(byteOffset)
					}

					// This is very similar to the interleaved reading code – can we unify these code paths?
					for i := 0; i < len(buf); i += dataSize {
						if i == 0 {
							if initialOffset > 0 {
								if _, err := r.Seek(int64(initialOffset), io.SeekCurrent); err != nil {
									yield(nil, fmt.Errorf("failed to seek to offset: %w", err))
									return
								}
							}
						} else {
							if _, err := r.Seek(stride, io.SeekCurrent); err != nil {
								yield(nil, fmt.Errorf("failed to seek to next value: %w", err))
								return
							}
						}

						var readLen int
						readLen, err = r.Read(buf[i : i+dataSize])
						if err != nil {
							break
						}
						n += readLen
					}
				}

				bytesRead += uint64(n)

				if err != nil {
					yield(nil, fmt.Errorf("failed to read data chunk: %w", err))
					return
				}

				// If we have plenty of data left in this chunk, we will have
				// read a value for every item in our batch. Otherwise, we may
				// have read only the number of elements left unread in the
				// chunk.
				//
				// For fixed-size, we can just do len(buf)/dataSize, but this
				// doesn't work for variable-size types.
				numValuesRead := min(batchSize, int(chunk.numValues)-valuesProcessed)

				for i := range numValuesRead {
					startIdx := int(i) * dataSize
					endIdx := int(i+1) * dataSize

					if ch.RawDataType == DataTypeString {
						// strOffsets should always have one more data point in
						// it than number of strings – we added the 0 at the
						// beginning and the last value is the end of the final
						// string.
						startIdx = int(strOffsets[i])
						endIdx = int(strOffsets[i+1])
					}

					val := interpret(buf[startIdx:endIdx], chunk.order)
					batch[i] = val.(TRaw)
				}

				valuesProcessed += numValuesRead

				// For strings, data size is 0 and we need to pull the
				// size of each individual string from the offsets at
				// the start of the chunk.

				// If scaling is disabled or not present, Scale() will just
				// return the input slice.
				scaledBatch, err := scaler.Scale(batch[:numValuesRead])
				if err != nil {
					yield(nil, fmt.Errorf("failed to scale batch: %w", err))
					return
				}

				if !yield(scaledBatch, nil) {
					return
				}
			}
		}
	}
}

func getInterpreter(dataType DataType) (func([]byte, binary.ByteOrder) any, error) {
	switch dataType {
	case DataTypeInt8:
		return func(b []byte, o binary.ByteOrder) any { return interpretInt8(b, o) }, nil
	case DataTypeInt16:
		return func(b []byte, o binary.ByteOrder) any { return interpretInt16(b, o) }, nil
	case DataTypeInt32:
		return func(b []byte, o binary.ByteOrder) any { return interpretInt32(b, o) }, nil
	case DataTypeInt64:
		return func(b []byte, o binary.ByteOrder) any { return interpretInt64(b, o) }, nil
	case DataTypeUint8:
		return func(b []byte, o binary.ByteOrder) any { return interpretUint8(b, o) }, nil
	case DataTypeUint16:
		return func(b []byte, o binary.ByteOrder) any { return interpretUint16(b, o) }, nil
	case DataTypeUint32:
		return func(b []byte, o binary.ByteOrder) any { return interpretUint32(b, o) }, nil
	case DataTypeUint64:
		return func(b []byte, o binary.ByteOrder) any { return interpretUint64(b, o) }, nil
	case DataTypeFloat32, DataTypeFloat32WithUnit:
		return func(b []byte, o binary.ByteOrder) any { return interpretFloat32(b, o) }, nil
	case DataTypeFloat64, DataTypeFloat64WithUnit:
		return func(b []byte, o binary.ByteOrder) any { return interpretFloat64(b, o) }, nil
	case DataTypeFloat128, DataTypeFloat128WithUnit:
		return func(b []byte, o binary.ByteOrder) any { return interpretFloat128(b, o) }, nil
	case DataTypeString:
		return func(b []byte, o binary.ByteOrder) any { return interpretString(b, o) }, nil
	case DataTypeBool:
		return func(b []byte, o binary.ByteOrder) any { return interpretBool(b, o) }, nil
	case DataTypeTimestamp:
		return func(b []byte, o binary.ByteOrder) any { return interpretTimestamp(b, o) }, nil
	case DataTypeComplex64:
		return func(b []byte, o binary.ByteOrder) any { return interpretComplex64(b, o) }, nil
	case DataTypeComplex128:
		return func(b []byte, o binary.ByteOrder) any { return interpretComplex128(b, o) }, nil
	default:
		return nil, fmt.Errorf("%w: data type %d", ErrUnsupportedType, dataType)
	}
}
