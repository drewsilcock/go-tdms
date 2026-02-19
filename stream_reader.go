// The stream reader allows iterative reading of values from a TDMS file for a
// particular channel.
//
// It uses batching to speed up reads, with functions that return either the
// batches as slices or the individual values. The stream reader that returns
// individual values still uses batching internally, it just helpfully unwraps
// the slice for you.

package tdms

import (
	"errors"
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
		interpret, err := getInterpreter(ch.RawDataType)
		if err != nil {
			yield(nil, fmt.Errorf("unable to get byte interpreter: %w", err))
			return
		}

		opts := renderReadOptions(options)

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

		var daqmxScaler *DAQmxScaler
		if ch.RawDataType == DataTypeDAQmxRawData {
			if opts.daqmxScaleIndex >= len(ch.scaler.scalers) {
				yield(nil, fmt.Errorf("invalid DAQmx scale index: %d", opts.daqmxScaleIndex))
				return
			}

			var ok bool
			daqmxScaler, ok = ch.scaler.scalers[opts.daqmxScaleIndex].(*DAQmxScaler)
			if !ok {
				yield(nil, fmt.Errorf("expected DAQmx scaler, got %T", daqmxScaler))
				return
			}
		}

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

			desiredSize := chunk.size
			if chunk.isDAQmx {
				// For DAQmx, we want to read to the end of the buffer that the
				// specified scaler is in, not the whole chunk.
				desiredSize = uint64(chunk.daqMXBufferSizes[daqmxScaler.rawBufferIndex])
			}

			for {
				// We don't want to read past the end of the chunk.
				bytesLeft := desiredSize - bytesRead
				if bytesLeft <= 0 {
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

				if chunk.isDAQmx {
					byteOffset := daqmxScaler.offsetWithinStride
					if daqmxScaler.scalerType == daqmxScalerTypeDigitalLine {
						// Digital line specifies offset in bits, not bytes.
						byteOffset = byteOffset / 8
					}

					stride := int64(chunk.daqMXBufferWidths[daqmxScaler.rawBufferIndex])
					initialOffset := int64(0)
					for i := range daqmxScaler.rawBufferIndex {
						initialOffset += int64(chunk.daqMXBufferSizes[i])
					}
					initialOffset += int64(byteOffset)

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

						readLen, err := r.Read(buf[i : i+dataSize])
						if err != nil {
							yield(nil, fmt.Errorf("failed to read data chunk: %w", err))
							return
						}
						n += readLen
					}
				} else if !chunk.isInterleaved {
					n, err = io.ReadFull(r, buf)
				} else {
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

					for i := 0; i < len(buf); i += dataSize {
						if i > 0 {
							if _, err := r.Seek(chunk.stride, io.SeekCurrent); err != nil {
								yield(nil, fmt.Errorf("failed to seek to initial offset: %w", err))
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

				if errors.Is(err, io.EOF) {
					break
				} else if err != nil {
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
