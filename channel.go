package tdms

import (
	"encoding/binary"
	"io"
	"iter"
	"time"
)

// Channel represents a data channel within a [Group]. Use the ReadData methods
// to access the channel's data in a type-safe manner.
type Channel struct {
	// Name is the name of this channel.
	Name string

	// GroupName is the name of the group that contains this channel.
	GroupName string

	// DataType is the type of data stored in this channel.
	DataType DataType

	// Properties contains all properties associated with this channel.
	Properties Properties

	reader         io.ReadSeeker
	path           string
	dataChunks     []dataChunk
	totalNumValues uint64
	file           *File
}

// dataChunk is similar to objectIndex, but is a single object index can
// correspond to multiple chunks whereas a single dataChunk instance corresponds
// to a single raw data chunk in the TDMS file.
//
// Note that a dataChunk instance is specific to an individual object, meaning a
// segment in a TDMS file with 2 channels and 3 chunks will have 6 dataChunk
// instances corresponding to it.
//
// This is purely for ease of use
// to make reading simpler and to keep all the necessary information self-contained.
type dataChunk struct {
	// offset is absolute from the start of the file
	offset        int64
	isInterleaved bool
	order         binary.ByteOrder
	size          uint64
	numValues     uint64
	stride        int64
}

// NumValues returns the total number of data values in this channel across all
// segments.
func (ch *Channel) NumValues() uint64 {
	return ch.totalNumValues
}

type readOptions struct {
	batchSize   int
	shouldScale bool
}

// ReadOption configures how data is read from a [Channel].
type ReadOption func(*readOptions)

// BatchSize sets the number of values read per batch during streaming. This
// controls the internal buffer size used by the streaming and batch readers.
func BatchSize(batchSize int) ReadOption {
	return func(opts *readOptions) {
		opts.batchSize = batchSize
	}
}

// WithScaling sets whether data should be scaled during streaming.
//
// Note that WithScaling() is equivalent to WithScaling(true).
//
// If [WithScaling] is not specified as an option, streaming will default to
// applying scaling.
func WithScaling(shouldScale ...bool) ReadOption {
	return func(opts *readOptions) {
		if len(shouldScale) > 0 {
			opts.shouldScale = shouldScale[0]
		} else {
			opts.shouldScale = true
		}
	}
}

// Data streaming functions that yield each item at a time.

// ReadInt8 returns an iterator that yields individual int8 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt8(options ...ReadOption) iter.Seq2[int8, error] {
	return func(yield func(int8, error) bool) {
		for value, err := range StreamReader(ch, options, interpretInt8) {
			if !yield(value.(int8), err) {
				return
			}
		}
	}
}

// ReadInt16 returns an iterator that yields individual int16 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt16(options ...ReadOption) iter.Seq2[int16, error] {
	return func(yield func(int16, error) bool) {
		for value, err := range StreamReader(ch, options, interpretInt16) {
			if !yield(value.(int16), err) {
				return
			}
		}
	}
}

// ReadInt32 returns an iterator that yields individual int32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt32(options ...ReadOption) iter.Seq2[int32, error] {
	return func(yield func(int32, error) bool) {
		for value, err := range StreamReader(ch, options, interpretInt32) {
			if !yield(value.(int32), err) {
				return
			}
		}
	}
}

// ReadInt64 returns an iterator that yields individual int64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt64(options ...ReadOption) iter.Seq2[int64, error] {
	return func(yield func(int64, error) bool) {
		for value, err := range StreamReader(ch, options, interpretInt64) {
			if !yield(value.(int64), err) {
				return
			}
		}
	}
}

// ReadUint8 returns an iterator that yields individual uint8 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint8(options ...ReadOption) iter.Seq2[uint8, error] {
	return func(yield func(uint8, error) bool) {
		for value, err := range StreamReader(ch, options, interpretUint8) {
			if !yield(value.(uint8), err) {
				return
			}
		}
	}
}

// ReadUint16 returns an iterator that yields individual uint16 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint16(options ...ReadOption) iter.Seq2[uint16, error] {
	return func(yield func(uint16, error) bool) {
		for value, err := range StreamReader(ch, options, interpretUint16) {
			if !yield(value.(uint16), err) {
				return
			}
		}
	}
}

// ReadUint32 returns an iterator that yields individual uint32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint32(options ...ReadOption) iter.Seq2[uint32, error] {
	return func(yield func(uint32, error) bool) {
		for value, err := range StreamReader(ch, options, interpretUint32) {
			if !yield(value.(uint32), err) {
				return
			}
		}
	}
}

// ReadUint64 returns an iterator that yields individual uint64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint64(options ...ReadOption) iter.Seq2[uint64, error] {
	return func(yield func(uint64, error) bool) {
		for value, err := range StreamReader(ch, options, interpretUint64) {
			if !yield(value.(uint64), err) {
				return
			}
		}
	}
}

// ReadFloat32 returns an iterator that yields individual float32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadFloat32(options ...ReadOption) iter.Seq2[float32, error] {
	return func(yield func(float32, error) bool) {
		for value, err := range StreamReader(ch, options, interpretFloat32) {
			if !yield(value.(float32), err) {
				return
			}
		}
	}
}

// ReadFloat64 returns an iterator that yields individual float64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadFloat64(options ...ReadOption) iter.Seq2[float64, error] {
	return func(yield func(float64, error) bool) {
		for value, err := range StreamReader(ch, options, interpretFloat64) {
			if !yield(value.(float64), err) {
				return
			}
		}
	}
}

// ReadFloat128 returns an iterator that yields individual [Float128] values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadFloat128(options ...ReadOption) iter.Seq2[Float128, error] {
	return func(yield func(Float128, error) bool) {
		for value, err := range StreamReader(ch, options, interpretFloat128) {
			if !yield(value.(Float128), err) {
				return
			}
		}
	}
}

// ReadString returns an iterator that yields individual string values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadString(options ...ReadOption) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for value, err := range StreamReader(ch, options, interpretString) {
			if !yield(value.(string), err) {
				return
			}
		}
	}
}

// ReadBool returns an iterator that yields individual bool values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadBool(options ...ReadOption) iter.Seq2[bool, error] {
	return func(yield func(bool, error) bool) {
		for value, err := range StreamReader(ch, options, interpretBool) {
			if !yield(value.(bool), err) {
				return
			}
		}
	}
}

// ReadTimestamp returns an iterator that yields individual [Timestamp] values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadTimestamp(options ...ReadOption) iter.Seq2[Timestamp, error] {
	return func(yield func(Timestamp, error) bool) {
		for value, err := range StreamReader(ch, options, interpretTimestamp) {
			if !yield(value.(Timestamp), err) {
				return
			}
		}
	}
}

// ReadTime returns an iterator that yields individual [time.Time] values from the channel.
// Timestamps are automatically converted from TDMS format. Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadTime(options ...ReadOption) iter.Seq2[time.Time, error] {
	return func(yield func(time.Time, error) bool) {
		for value, err := range StreamReader(ch, options, interpretTime) {
			if !yield(value.(time.Time), err) {
				return
			}
		}
	}
}

// ReadComplex64 returns an iterator that yields individual complex64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadComplex64(options ...ReadOption) iter.Seq2[complex64, error] {
	return func(yield func(complex64, error) bool) {
		for value, err := range StreamReader(ch, options, interpretComplex64) {
			if !yield(value.(complex64), err) {
				return
			}
		}
	}
}

// ReadComplex128 returns an iterator that yields individual complex128 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadComplex128(options ...ReadOption) iter.Seq2[complex128, error] {
	return func(yield func(complex128, error) bool) {
		for value, err := range StreamReader(ch, options, interpretComplex128) {
			if !yield(value.(complex128), err) {
				return
			}
		}
	}
}

// Data streaming functions that yield items in batches.

// ReadInt8Batch returns an iterator that yields batches of int8 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt8Batch(options ...ReadOption) iter.Seq2[[]int8, error] {
	return func(yield func([]int8, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretInt8) {
			if !yield(value.([]int8), err) {
				return
			}
		}
	}
}

// ReadInt16Batch returns an iterator that yields batches of int16 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt16Batch(options ...ReadOption) iter.Seq2[[]int16, error] {
	return func(yield func([]int16, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretInt16) {
			if !yield(value.([]int16), err) {
				return
			}
		}
	}
}

// ReadInt32Batch returns an iterator that yields batches of int32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt32Batch(options ...ReadOption) iter.Seq2[[]int32, error] {
	return func(yield func([]int32, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretInt32) {
			if !yield(value.([]int32), err) {
				return
			}
		}
	}
}

// ReadInt64Batch returns an iterator that yields batches of int64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt64Batch(options ...ReadOption) iter.Seq2[[]int64, error] {
	return func(yield func([]int64, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretInt64) {
			if !yield(value.([]int64), err) {
				return
			}
		}
	}
}

// ReadUint8Batch returns an iterator that yields batches of uint8 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint8Batch(options ...ReadOption) iter.Seq2[[]uint8, error] {
	return func(yield func([]uint8, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretUint8) {
			if !yield(value.([]uint8), err) {
				return
			}
		}
	}
}

// ReadUint16Batch returns an iterator that yields batches of uint16 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint16Batch(options ...ReadOption) iter.Seq2[[]uint16, error] {
	return func(yield func([]uint16, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretUint16) {
			if !yield(value.([]uint16), err) {
				return
			}
		}
	}
}

// ReadUint32Batch returns an iterator that yields batches of uint32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint32Batch(options ...ReadOption) iter.Seq2[[]uint32, error] {
	return func(yield func([]uint32, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretUint32) {
			if !yield(value.([]uint32), err) {
				return
			}
		}
	}
}

// ReadUint64Batch returns an iterator that yields batches of uint64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint64Batch(options ...ReadOption) iter.Seq2[[]uint64, error] {
	return func(yield func([]uint64, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretUint64) {
			if !yield(value.([]uint64), err) {
				return
			}
		}
	}
}

// ReadFloat32Batch returns an iterator that yields batches of float32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadFloat32Batch(options ...ReadOption) iter.Seq2[[]float32, error] {
	return func(yield func([]float32, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretFloat32) {
			if !yield(value.([]float32), err) {
				return
			}
		}
	}
}

// ReadFloat64Batch returns an iterator that yields batches of float64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadFloat64Batch(options ...ReadOption) iter.Seq2[[]float64, error] {
	return func(yield func([]float64, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretFloat64) {
			if !yield(value.([]float64), err) {
				return
			}
		}
	}
}

// ReadFloat128Batch returns an iterator that yields batches of [Float128] values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadFloat128Batch(options ...ReadOption) iter.Seq2[[]Float128, error] {
	return func(yield func([]Float128, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretFloat128) {
			if !yield(value.([]Float128), err) {
				return
			}
		}
	}
}

// ReadStringBatch returns an iterator that yields batches of string values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadStringBatch(options ...ReadOption) iter.Seq2[[]string, error] {
	return func(yield func([]string, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretString) {
			if !yield(value.([]string), err) {
				return
			}
		}
	}
}

// ReadBoolBatch returns an iterator that yields batches of bool values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadBoolBatch(options ...ReadOption) iter.Seq2[[]bool, error] {
	return func(yield func([]bool, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretBool) {
			if !yield(value.([]bool), err) {
				return
			}
		}
	}
}

// ReadTimestampBatch returns an iterator that yields batches of [Timestamp] values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadTimestampBatch(options ...ReadOption) iter.Seq2[[]Timestamp, error] {
	return func(yield func([]Timestamp, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretTimestamp) {
			if !yield(value.([]Timestamp), err) {
				return
			}
		}
	}
}

// ReadTimeBatch returns an iterator that yields batches of [time.Time] values from the channel.
// Timestamps are automatically converted from TDMS format. Use BatchSize option to control batch size.
func (ch *Channel) ReadTimeBatch(options ...ReadOption) iter.Seq2[[]time.Time, error] {
	return func(yield func([]time.Time, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretTime) {
			if !yield(value.([]time.Time), err) {
				return
			}
		}
	}
}

// ReadComplex64Batch returns an iterator that yields batches of complex64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadComplex64Batch(options ...ReadOption) iter.Seq2[[]complex64, error] {
	return func(yield func([]complex64, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretComplex64) {
			if !yield(value.([]complex64), err) {
				return
			}
		}
	}
}

// ReadComplex128Batch returns an iterator that yields batches of complex128 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadComplex128Batch(options ...ReadOption) iter.Seq2[[]complex128, error] {
	return func(yield func([]complex128, error) bool) {
		for value, err := range BatchStreamReader(ch, options, interpretComplex128) {
			if !yield(value.([]complex128), err) {
				return
			}
		}
	}
}

// Data streaming functions that read all the data for a channel in one go.

// ReadInt8All reads all int8 values from the channel into a single slice.
func (ch *Channel) ReadInt8All(options ...ReadOption) ([]int8, error) {
	values, err := readAllData(ch, options, interpretInt8)
	return values.([]int8), err
}

// ReadInt16All reads all int16 values from the channel into a single slice.
func (ch *Channel) ReadInt16All(options ...ReadOption) ([]int16, error) {
	values, err := readAllData(ch, options, interpretInt16)
	return values.([]int16), err
}

// ReadInt32All reads all int32 values from the channel into a single slice.
func (ch *Channel) ReadInt32All(options ...ReadOption) ([]int32, error) {
	values, err := readAllData(ch, options, interpretInt32)
	return values.([]int32), err
}

// ReadInt64All reads all int64 values from the channel into a single slice.
func (ch *Channel) ReadInt64All(options ...ReadOption) ([]int64, error) {
	values, err := readAllData(ch, options, interpretInt64)
	return values.([]int64), err
}

// ReadUint8All reads all uint8 values from the channel into a single slice.
func (ch *Channel) ReadUint8All(options ...ReadOption) ([]uint8, error) {
	values, err := readAllData(ch, options, interpretUint8)
	return values.([]uint8), err
}

// ReadUint16All reads all uint16 values from the channel into a single slice.
func (ch *Channel) ReadUint16All(options ...ReadOption) ([]uint16, error) {
	values, err := readAllData(ch, options, interpretUint16)
	return values.([]uint16), err
}

// ReadUint32All reads all uint32 values from the channel into a single slice.
func (ch *Channel) ReadUint32All(options ...ReadOption) ([]uint32, error) {
	values, err := readAllData(ch, options, interpretUint32)
	return values.([]uint32), err
}

// ReadUint64All reads all uint64 values from the channel into a single slice.
func (ch *Channel) ReadUint64All(options ...ReadOption) ([]uint64, error) {
	values, err := readAllData(ch, options, interpretUint64)
	return values.([]uint64), err
}

// ReadFloat32All reads all float32 values from the channel into a single slice.
func (ch *Channel) ReadFloat32All(options ...ReadOption) ([]float32, error) {
	values, err := readAllData(ch, options, interpretFloat32)
	return values.([]float32), err
}

// ReadFloat64All reads all float64 values from the channel into a single slice.
func (ch *Channel) ReadFloat64All(options ...ReadOption) ([]float64, error) {
	values, err := readAllData(ch, options, interpretFloat64)
	return values.([]float64), err
}

// ReadFloat128All reads all [Float128] values from the channel into a single slice.
func (ch *Channel) ReadFloat128All(options ...ReadOption) ([]Float128, error) {
	values, err := readAllData(ch, options, interpretFloat128)
	return values.([]Float128), err
}

// ReadStringAll reads all string values from the channel into a single slice.
func (ch *Channel) ReadStringAll(options ...ReadOption) ([]string, error) {
	values, err := readAllData(ch, options, interpretString)
	return values.([]string), err
}

// ReadBoolAll reads all bool values from the channel into a single slice.
func (ch *Channel) ReadBoolAll(options ...ReadOption) ([]bool, error) {
	values, err := readAllData(ch, options, interpretBool)
	return values.([]bool), err
}

// ReadTimestampAll reads all [Timestamp] values from the channel into a single slice.
func (ch *Channel) ReadTimestampAll(options ...ReadOption) ([]Timestamp, error) {
	values, err := readAllData(ch, options, interpretTimestamp)
	return values.([]Timestamp), err
}

// ReadTimeAll reads all [time.Time] values from the channel into a single slice.
// Timestamps are automatically converted from TDMS format.
func (ch *Channel) ReadTimeAll(options ...ReadOption) ([]time.Time, error) {
	values, err := readAllData(ch, options, interpretTime)
	return values.([]time.Time), err
}

// ReadComplex64All reads all complex64 values from the channel into a single slice.
func (ch *Channel) ReadComplex64All(options ...ReadOption) ([]complex64, error) {
	values, err := readAllData(ch, options, interpretComplex64)
	return values.([]complex64), err
}

// ReadComplex128All reads all complex128 values from the channel into a single slice.
func (ch *Channel) ReadComplex128All(options ...ReadOption) ([]complex128, error) {
	values, err := readAllData(ch, options, interpretComplex128)
	return values.([]complex128), err
}
