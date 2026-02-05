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
	"io"
	"iter"
)

type interpreter[T any] func([]byte, binary.ByteOrder) T

// StreamReader still internally uses batching, hence the batch size param,
// however it returns the results as individual values, which may be more useful
// in many scenarios.
func StreamReader[T any](
	ch *Channel,
	batchSize int,
	dataType DataType,
	interpret interpreter[T],
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for batch, err := range BatchStreamReader(ch, batchSize, dataType, interpret) {
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
	batchSize int,
	dataType DataType,
	interpret interpreter[T],
) iter.Seq2[[]T, error] {
	return func(yield func([]T, error) bool) {
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
			var strOffsets []uint32
			if dataType == DataTypeString {
				strOffsetsBytes := make([]byte, chunk.numValues)
				if n, err := r.Read(strOffsetsBytes); err != nil {
					yield(nil, err)
					return
				} else {
					bytesRead += uint64(n)
				}

				strOffsets = make([]uint32, chunk.numValues)
				for i := range chunk.numValues {
					strOffsets[i] = chunk.order.Uint32(strOffsetsBytes[i*4:])
				}
			}

			// For strings, we need to keep track of the current index that
			// we're processing so that we can get the offset for that value.
			valuesProcessed := 0

			for {
				// We don't want to read past the end of the chunk.
				bytesLeft := chunk.chunkSize - bytesRead
				if bufLen > bytesLeft {
					// This retains capacity.
					buf = buf[:bytesLeft]
				} else {
					buf = buf[:bufLen]
				}

				n := 0
				var err error
				if chunk.isInterleaved == false {
					n, err = io.ReadFull(r, buf)
				} else {

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
						startIdx = int(strOffsets[i])
						if int(i+1) < len(strOffsets) {
							endIdx = int(strOffsets[i+1])
						} else {
							endIdx = len(buf)
						}
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
