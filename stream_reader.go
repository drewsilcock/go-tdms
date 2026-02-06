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

// StreamReader still internally uses batching, hence the batch size param,
// however it returns the results as individual values, which may be more useful
// in many scenarios.
func StreamReader[T any](
	ch *Channel,
	options []ReadOption,
	dataType DataType,
	interpret interpreter[T],
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for batch, err := range BatchStreamReader(ch, options, dataType, interpret) {
			if err != nil {
				yield(*new(T), err)
				return
			}

			for _, datum := range batch {
				if !yield(datum, nil) {
					return
				}
			}
		}
	}
}

// Be aware that this re-uses the same batch during the lifetime of the
// iterator. If you want to collect all the data from the BatchStreamReader, you
// need to copy the data into your own buffer.
//
// We could also implement `ReadAll()` functions for this purpose, or we could
// pass in the buffer externally to make it explicit that this is happening.
//
// The approach used here to convert bytes to T is likely to mean that the
// interpret functions can't be inlined, but I shouldn't think this would make a
// big difference. It would be interesting to benchmark that.
//
// TODO: This doesn't correctly handle reading channels of type string.
func BatchStreamReader[T any](
	ch *Channel,
	options []ReadOption,
	dataType DataType,
	interpret interpreter[T],
) iter.Seq2[[]T, error] {
	return func(yield func([]T, error) bool) {
		opts := readOptions{}
		for _, opt := range options {
			opt(&opts)
		}

		if opts.batchSize == 0 {
			opts.batchSize = 2056
			if dataType == DataTypeString {
				// Strings are generally much larger than individual ints or
				// floats, so we use much smaller default batch size.
				opts.batchSize = 256
			}
		}

		// If we have fewer data points in total than a single batch size, we
		// can allocate only what we need.
		batchSize := min(opts.batchSize, int(ch.totalNumValues))
		dataSize := dataType.Size()

		buf := make([]byte, batchSize*dataSize)
		bufLen := uint64(len(buf))
		batch := make([]T, batchSize)
		r := ch.f.f

		for _, chunk := range ch.dataChunks {
			if _, err := r.Seek(chunk.offset, io.SeekStart); err != nil {
				yield(nil, err)
				return
			}

			bytesRead := uint64(0)

			// Special case for strings, where the indices into the strings are
			// stored at the beginning of the chunk.
			strOffsets := []uint32{0}
			if dataType == DataTypeString {
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
				if dataType == DataTypeString {
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

					if dataType == DataTypeString {
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
				// size of each individual string from the offsetes at
				// the start of the chunk.

				if !yield(batch[:numValuesRead], nil) {
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
func readAllData[T any](ch *Channel, options []ReadOption, dataType DataType, interpret interpreter[T]) ([]T, error) {
	values := make([]T, 0, ch.totalNumValues)

	for batch, err := range BatchStreamReader(ch, options, dataType, interpret) {
		if err != nil {
			return nil, err
		}

		values = append(values, batch...)
	}

	return values, nil
}
