// The stream reader allows iterative reading of values from a TDMS file for a
// particular channel.
//
// It uses batching to speed up reads, with functions that return either the
// batches as slices or the individual values. The stream reader that returns
// individual values still uses batching internally, it just helpfully unwraps
// the slice for you.
//
// TODO: Handle scaling.

package tdms

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
)

type interpreter[T any] func([]byte, binary.ByteOrder) T

// StreamReader returns an iterator yielding individual values from the channel.
//
// It internally uses batching for performance, but unwraps the batches
// to yield one value at a time. Use the [BatchSize] option to control the
// internal buffer size. This is the most convenient way to iterate over channel
// data when you need to process values individually.
func StreamReader[T any](ch *Channel, options []ReadOption, interpret interpreter[T]) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		for batch, err := range BatchStreamReader(ch, options, interpret) {
			if err != nil {
				yield(nil, err)
				return
			}

			// This would be a lot shorter using reflect.ValueOf(), but it's a
			// relatively hot path so I'm avoiding reflection with a type
			// switch. If you think this is bad, you should see the type
			// promotion code.
			switch v := batch.(type) {
			case []int8:
				if !yieldSlice(v, yield) {
					return
				}
			case []int16:
				if !yieldSlice(v, yield) {
					return
				}
			case []int32:
				if !yieldSlice(v, yield) {
					return
				}
			case []int64:
				if !yieldSlice(v, yield) {
					return
				}
			case []uint8:
				if !yieldSlice(v, yield) {
					return
				}
			case []uint16:
				if !yieldSlice(v, yield) {
					return
				}
			case []uint32:
				if !yieldSlice(v, yield) {
					return
				}
			case []uint64:
				if !yieldSlice(v, yield) {
					return
				}
			case []float32:
				if !yieldSlice(v, yield) {
					return
				}
			case []float64:
				if !yieldSlice(v, yield) {
					return
				}
			case []Float128:
				if !yieldSlice(v, yield) {
					return
				}
			case []string:
				if !yieldSlice(v, yield) {
					return
				}
			case []bool:
				if !yieldSlice(v, yield) {
					return
				}
			case []Timestamp:
				if !yieldSlice(v, yield) {
					return
				}
			case []complex64:
				if !yieldSlice(v, yield) {
					return
				}
			case []complex128:
				if !yieldSlice(v, yield) {
					return
				}
			}
		}
	}
}

func yieldSlice[T any](values []T, yield func(any, error) bool) bool {
	for i := range values {
		if !yield(values[i], nil) {
			return false
		}
	}

	return true
}

// BatchStreamReader returns an iterator that yields batches of values from the
// channel. Each batch is a slice of values read from the underlying file. Use
// the [BatchSize] option to control how many values are read in each batch.
//
// Important: The same underlying slice is reused for each batch to avoid
// allocations. If you need to retain batch data beyond the current iteration,
// you must copy it to your own buffer. For reading all data into a single
// slice, use the ReadData*All methods on [Channel] instead.
func BatchStreamReader[T any](ch *Channel, options []ReadOption, interpret interpreter[T]) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		opts := readOptions{
			batchSize:   0,
			shouldScale: true,
		}
		for _, opt := range options {
			opt(&opts)
		}

		if opts.batchSize == 0 {
			opts.batchSize = 2056
			if ch.DataType == DataTypeString {
				// Strings are generally much larger than individual ints or
				// floats, so we use much smaller default batch size.
				opts.batchSize = 256
			}
		}

		// If we have fewer data points in total than a single batch size, we
		// can allocate only what we need.
		batchSize := min(opts.batchSize, int(ch.totalNumValues))
		dataSize := ch.DataType.Size()

		var scaler *Multiscaler
		if opts.shouldScale {
			var err error
			scaler, err = getChannelScaler(ch, batchSize)
			if err != nil {
				yield(nil, err)
				return
			}
		}

		buf := make([]byte, batchSize*dataSize)
		bufLen := uint64(len(buf))
		batch := make([]T, batchSize)
		r := ch.reader

		for _, chunk := range ch.dataChunks {
			if _, err := r.Seek(chunk.offset, io.SeekStart); err != nil {
				yield(nil, err)
				return
			}

			bytesRead := uint64(0)

			// Special case for strings, where the indices into the strings are
			// stored at the beginning of the chunk.
			strOffsets := []uint32{0}
			if ch.DataType == DataTypeString {
				strOffsetsBytes := make([]byte, chunk.numValues*4)
				if n, err := r.Read(strOffsetsBytes); err != nil {
					yield(nil, err)
					return
				} else {
					bytesRead += uint64(n)
				}

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

				// For strings, our buf starts with length 0 because data size
				// is 0. Now that we know how long each value is, we can make
				// buf big enough to hold the values for this batch.
				if ch.DataType == DataTypeString {
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
				if !chunk.isInterleaved {
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
								yield(nil, err)
								return
							}
						}

						if readLen, err := r.Read(buf[int(i)*dataSize : int(i+1)*dataSize]); err != nil {
							yield(nil, err)
							return
						} else {
							n += readLen
						}
					}
				}

				bytesRead += uint64(n)

				// If the final batch doesn't line up with the end of the chunk,
				// we will get unexpected EOF. If our penultimate batch does
				// exactly line up with the end of the chunk, we will get EOF
				// when we try to read the next batch where there's no data
				// left.
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					// We've reached the end of the chunk.
					break
				}

				if err != nil {
					yield(nil, err)
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

					if ch.DataType == DataTypeString {
						// strOffsets should always have one more data point in
						// it than number of strings – we added the 0 at the
						// beginning and the last value is the end of the final
						// string.
						startIdx = int(strOffsets[i])
						endIdx = int(strOffsets[i+1])
					}

					batch[i] = interpret(buf[startIdx:endIdx], chunk.order)
				}

				valuesProcessed += numValuesRead

				// For strings, data size is 0 and we need to pull the
				// size of each individual string from the offsets at
				// the start of the chunk.

				if scaler != nil {
					scaledBatch, err := scaler.Scale(batch[:numValuesRead])
					if err != nil {
						yield(nil, err)
						return
					}

					if !yield(scaledBatch, nil) {
						return
					}
				} else if !yield(batch[:numValuesRead], nil) {
					return
				}
			}
		}
	}
}

// readAllData reads all data from a channel and put it into a single slice.
//
// By re-using BatchStreamReader here, we can avoid having to allocate 2*N bytes
// – one for the raw bytes and other for the interpreted values. The raw bytes
// are still batched while we allocate the values slice up-front. It's also
// cleaner in terms of the code as we avoid re-implementing the underlying read
// functionality.
func readAllData[T any](ch *Channel, options []ReadOption, interpret interpreter[T]) (any, error) {
	// Remember, scaling can change the type so that it's not the same as what's
	// set in the channel metadata.
	var values any

	for batch, err := range BatchStreamReader(ch, options, interpret) {
		if err != nil {
			return nil, err
		}

		switch v := batch.(type) {
		case []int8:
			if values == nil {
				values = make([]int8, 0, ch.totalNumValues)
			}

			values = append(values.([]int8), v...)
		case []int16:
			if values == nil {
				values = make([]int16, 0, ch.totalNumValues)
			}

			values = append(values.([]int16), v...)
		case []int32:
			if values == nil {
				values = make([]int32, 0, ch.totalNumValues)
			}

			values = append(values.([]int32), v...)
		case []int64:
			if values == nil {
				values = make([]int64, 0, ch.totalNumValues)
			}

			values = append(values.([]int64), v...)
		case []uint8:
			if values == nil {
				values = make([]uint8, 0, ch.totalNumValues)
			}

			values = append(values.([]uint8), v...)
		case []uint16:
			if values == nil {
				values = make([]uint16, 0, ch.totalNumValues)
			}

			values = append(values.([]uint16), v...)
		case []uint32:
			if values == nil {
				values = make([]uint32, 0, ch.totalNumValues)
			}

			values = append(values.([]uint32), v...)
		case []uint64:
			if values == nil {
				values = make([]uint64, 0, ch.totalNumValues)
			}

			values = append(values.([]uint64), v...)
		case []float32:
			if values == nil {
				values = make([]float32, 0, ch.totalNumValues)
			}

			values = append(values.([]float32), v...)
		case []float64:
			if values == nil {
				values = make([]float64, 0, ch.totalNumValues)
			}

			values = append(values.([]float64), v...)
		case []Float128:
			if values == nil {
				values = make([]Float128, 0, ch.totalNumValues)
			}

			values = append(values.([]Float128), v...)
		case []string:
			if values == nil {
				values = make([]string, 0, ch.totalNumValues)
			}

			values = append(values.([]string), v...)
		case []bool:
			if values == nil {
				values = make([]bool, 0, ch.totalNumValues)
			}

			values = append(values.([]bool), v...)
		case []Timestamp:
			if values == nil {
				values = make([]Timestamp, 0, ch.totalNumValues)
			}

			values = append(values.([]Timestamp), v...)
		case []complex64:
			if values == nil {
				values = make([]complex64, 0, ch.totalNumValues)
			}

			values = append(values.([]complex64), v...)
		case []complex128:
			if values == nil {
				values = make([]complex128, 0, ch.totalNumValues)
			}

			values = append(values.([]complex128), v...)
		default:
			return nil, ErrUnsupportedType
		}
	}

	return values, nil
}
