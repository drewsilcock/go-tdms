package tdms

import (
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
	Properties map[string]Property

	f              *File
	path           string
	dataChunks     []dataChunk
	totalNumValues uint64
}

// Group returns the [Group] that this channel belongs to.
func (ch *Channel) Group() Group {
	return ch.f.Groups[ch.GroupName]
}

// NumValues returns the total number of data values in this channel across all
// segments.
func (ch *Channel) NumValues() uint64 {
	return ch.totalNumValues
}

type readOptions struct {
	batchSize int
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

// Data streaming functions that yield each item at a time.

// ReadDataAsInt8 returns an iterator that yields individual int8 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsInt8(options ...ReadOption) iter.Seq2[int8, error] {
	return StreamReader(ch, options, DataTypeInt8, interpretInt8)
}

// ReadDataAsInt16 returns an iterator that yields individual int16 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsInt16(options ...ReadOption) iter.Seq2[int16, error] {
	return StreamReader(ch, options, DataTypeInt16, interpretInt16)
}

// ReadDataAsInt32 returns an iterator that yields individual int32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsInt32(options ...ReadOption) iter.Seq2[int32, error] {
	return StreamReader(ch, options, DataTypeInt32, interpretInt32)
}

// ReadDataAsInt64 returns an iterator that yields individual int64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsInt64(options ...ReadOption) iter.Seq2[int64, error] {
	return StreamReader(ch, options, DataTypeInt64, interpretInt64)
}

// ReadDataAsUint8 returns an iterator that yields individual uint8 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsUint8(options ...ReadOption) iter.Seq2[uint8, error] {
	return StreamReader(ch, options, DataTypeUint8, interpretUint8)
}

// ReadDataAsUint16 returns an iterator that yields individual uint16 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsUint16(options ...ReadOption) iter.Seq2[uint16, error] {
	return StreamReader(ch, options, DataTypeUint16, interpretUint16)
}

// ReadDataAsUint32 returns an iterator that yields individual uint32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsUint32(options ...ReadOption) iter.Seq2[uint32, error] {
	return StreamReader(ch, options, DataTypeUint32, interpretUint32)
}

// ReadDataAsUint64 returns an iterator that yields individual uint64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsUint64(options ...ReadOption) iter.Seq2[uint64, error] {
	return StreamReader(ch, options, DataTypeUint64, interpretUint64)
}

// ReadDataAsFloat32 returns an iterator that yields individual float32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsFloat32(options ...ReadOption) iter.Seq2[float32, error] {
	return StreamReader(ch, options, DataTypeFloat32, interpretFloat32)
}

// ReadDataAsFloat64 returns an iterator that yields individual float64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsFloat64(options ...ReadOption) iter.Seq2[float64, error] {
	return StreamReader(ch, options, DataTypeFloat64, interpretFloat64)
}

// ReadDataAsFloat128 returns an iterator that yields individual [Float128] values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsFloat128(options ...ReadOption) iter.Seq2[Float128, error] {
	return StreamReader(ch, options, DataTypeFloat128, interpretFloat128)
}

// ReadDataAsString returns an iterator that yields individual string values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsString(options ...ReadOption) iter.Seq2[string, error] {
	return StreamReader(ch, options, DataTypeString, interpretString)
}

// ReadDataAsBool returns an iterator that yields individual bool values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsBool(options ...ReadOption) iter.Seq2[bool, error] {
	return StreamReader(ch, options, DataTypeBool, interpretBool)
}

// ReadDataAsTimestamp returns an iterator that yields individual [Timestamp] values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsTimestamp(options ...ReadOption) iter.Seq2[Timestamp, error] {
	return StreamReader(ch, options, DataTypeTimestamp, interpretTimestamp)
}

// ReadDataAsTime returns an iterator that yields individual [time.Time] values from the channel.
// Timestamps are automatically converted from TDMS format. Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsTime(options ...ReadOption) iter.Seq2[time.Time, error] {
	return StreamReader(ch, options, DataTypeTimestamp, interpretTime)
}

// ReadDataAsComplex64 returns an iterator that yields individual complex64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsComplex64(options ...ReadOption) iter.Seq2[complex64, error] {
	return StreamReader(ch, options, DataTypeComplex64, interpretComplex64)
}

// ReadDataAsComplex128 returns an iterator that yields individual complex128 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadDataAsComplex128(options ...ReadOption) iter.Seq2[complex128, error] {
	return StreamReader(ch, options, DataTypeComplex128, interpretComplex128)
}

// Data streaming functions that yield items in batches.

// ReadDataAsInt8Batch returns an iterator that yields batches of int8 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsInt8Batch(options ...ReadOption) iter.Seq2[[]int8, error] {
	return BatchStreamReader(ch, options, DataTypeInt8, interpretInt8)
}

// ReadDataAsInt16Batch returns an iterator that yields batches of int16 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsInt16Batch(options ...ReadOption) iter.Seq2[[]int16, error] {
	return BatchStreamReader(ch, options, DataTypeInt16, interpretInt16)
}

// ReadDataAsInt32Batch returns an iterator that yields batches of int32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsInt32Batch(options ...ReadOption) iter.Seq2[[]int32, error] {
	return BatchStreamReader(ch, options, DataTypeInt32, interpretInt32)
}

// ReadDataAsInt64Batch returns an iterator that yields batches of int64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsInt64Batch(options ...ReadOption) iter.Seq2[[]int64, error] {
	return BatchStreamReader(ch, options, DataTypeInt64, interpretInt64)
}

// ReadDataAsUint8Batch returns an iterator that yields batches of uint8 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsUint8Batch(options ...ReadOption) iter.Seq2[[]uint8, error] {
	return BatchStreamReader(ch, options, DataTypeUint8, interpretUint8)
}

// ReadDataAsUint16Batch returns an iterator that yields batches of uint16 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsUint16Batch(options ...ReadOption) iter.Seq2[[]uint16, error] {
	return BatchStreamReader(ch, options, DataTypeUint16, interpretUint16)
}

// ReadDataAsUint32Batch returns an iterator that yields batches of uint32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsUint32Batch(options ...ReadOption) iter.Seq2[[]uint32, error] {
	return BatchStreamReader(ch, options, DataTypeUint32, interpretUint32)
}

// ReadDataAsUint64Batch returns an iterator that yields batches of uint64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsUint64Batch(options ...ReadOption) iter.Seq2[[]uint64, error] {
	return BatchStreamReader(ch, options, DataTypeUint64, interpretUint64)
}

// ReadDataAsFloat32Batch returns an iterator that yields batches of float32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsFloat32Batch(options ...ReadOption) iter.Seq2[[]float32, error] {
	return BatchStreamReader(ch, options, DataTypeFloat32, interpretFloat32)
}

// ReadDataAsFloat64Batch returns an iterator that yields batches of float64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsFloat64Batch(options ...ReadOption) iter.Seq2[[]float64, error] {
	return BatchStreamReader(ch, options, DataTypeFloat64, interpretFloat64)
}

// ReadDataAsFloat128Batch returns an iterator that yields batches of [Float128] values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsFloat128Batch(options ...ReadOption) iter.Seq2[[]Float128, error] {
	return BatchStreamReader(ch, options, DataTypeFloat128, interpretFloat128)
}

// ReadDataAsStringBatch returns an iterator that yields batches of string values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsStringBatch(options ...ReadOption) iter.Seq2[[]string, error] {
	return BatchStreamReader(ch, options, DataTypeString, interpretString)
}

// ReadDataAsBoolBatch returns an iterator that yields batches of bool values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsBoolBatch(options ...ReadOption) iter.Seq2[[]bool, error] {
	return BatchStreamReader(ch, options, DataTypeBool, interpretBool)
}

// ReadDataAsTimestampBatch returns an iterator that yields batches of [Timestamp] values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsTimestampBatch(options ...ReadOption) iter.Seq2[[]Timestamp, error] {
	return BatchStreamReader(ch, options, DataTypeTimestamp, interpretTimestamp)
}

// ReadDataAsTimeBatch returns an iterator that yields batches of [time.Time] values from the channel.
// Timestamps are automatically converted from TDMS format. Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsTimeBatch(options ...ReadOption) iter.Seq2[[]time.Time, error] {
	return BatchStreamReader(ch, options, DataTypeTimestamp, interpretTime)
}

// ReadDataAsComplex64Batch returns an iterator that yields batches of complex64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsComplex64Batch(options ...ReadOption) iter.Seq2[[]complex64, error] {
	return BatchStreamReader(ch, options, DataTypeComplex64, interpretComplex64)
}

// ReadDataAsComplex128Batch returns an iterator that yields batches of complex128 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadDataAsComplex128Batch(options ...ReadOption) iter.Seq2[[]complex128, error] {
	return BatchStreamReader(ch, options, DataTypeComplex128, interpretComplex128)
}

// Data streaming functions that read all the data for a channel in one go.

// ReadDataInt8All reads all int8 values from the channel into a single slice.
func (ch *Channel) ReadDataInt8All(options ...ReadOption) ([]int8, error) {
	return readAllData(ch, options, DataTypeInt8, interpretInt8)
}

// ReadDataInt16All reads all int16 values from the channel into a single slice.
func (ch *Channel) ReadDataInt16All(options ...ReadOption) ([]int16, error) {
	return readAllData(ch, options, DataTypeInt16, interpretInt16)
}

// ReadDataInt32All reads all int32 values from the channel into a single slice.
func (ch *Channel) ReadDataInt32All(options ...ReadOption) ([]int32, error) {
	return readAllData(ch, options, DataTypeInt32, interpretInt32)
}

// ReadDataInt64All reads all int64 values from the channel into a single slice.
func (ch *Channel) ReadDataInt64All(options ...ReadOption) ([]int64, error) {
	return readAllData(ch, options, DataTypeInt64, interpretInt64)
}

// ReadDataUint8All reads all uint8 values from the channel into a single slice.
func (ch *Channel) ReadDataUint8All(options ...ReadOption) ([]uint8, error) {
	return readAllData(ch, options, DataTypeUint8, interpretUint8)
}

// ReadDataUint16All reads all uint16 values from the channel into a single slice.
func (ch *Channel) ReadDataUint16All(options ...ReadOption) ([]uint16, error) {
	return readAllData(ch, options, DataTypeUint16, interpretUint16)
}

// ReadDataUint32All reads all uint32 values from the channel into a single slice.
func (ch *Channel) ReadDataUint32All(options ...ReadOption) ([]uint32, error) {
	return readAllData(ch, options, DataTypeUint32, interpretUint32)
}

// ReadDataUint64All reads all uint64 values from the channel into a single slice.
func (ch *Channel) ReadDataUint64All(options ...ReadOption) ([]uint64, error) {
	return readAllData(ch, options, DataTypeUint64, interpretUint64)
}

// ReadDataFloat32All reads all float32 values from the channel into a single slice.
func (ch *Channel) ReadDataFloat32All(options ...ReadOption) ([]float32, error) {
	return readAllData(ch, options, DataTypeFloat32, interpretFloat32)
}

// ReadDataFloat64All reads all float64 values from the channel into a single slice.
func (ch *Channel) ReadDataFloat64All(options ...ReadOption) ([]float64, error) {
	return readAllData(ch, options, DataTypeFloat64, interpretFloat64)
}

// ReadDataFloat128All reads all [Float128] values from the channel into a single slice.
func (ch *Channel) ReadDataFloat128All(options ...ReadOption) ([]Float128, error) {
	return readAllData(ch, options, DataTypeFloat128, interpretFloat128)
}

// ReadDataStringAll reads all string values from the channel into a single slice.
func (ch *Channel) ReadDataStringAll(options ...ReadOption) ([]string, error) {
	return readAllData(ch, options, DataTypeString, interpretString)
}

// ReadDataBoolAll reads all bool values from the channel into a single slice.
func (ch *Channel) ReadDataBoolAll(options ...ReadOption) ([]bool, error) {
	return readAllData(ch, options, DataTypeBool, interpretBool)
}

// ReadDataTimestampAll reads all [Timestamp] values from the channel into a single slice.
func (ch *Channel) ReadDataTimestampAll(options ...ReadOption) ([]Timestamp, error) {
	return readAllData(ch, options, DataTypeTimestamp, interpretTimestamp)
}

// ReadDataTimeAll reads all [time.Time] values from the channel into a single slice.
// Timestamps are automatically converted from TDMS format.
func (ch *Channel) ReadDataTimeAll(options ...ReadOption) ([]time.Time, error) {
	return readAllData(ch, options, DataTypeTimestamp, interpretTime)
}

// ReadDataComplex64All reads all complex64 values from the channel into a single slice.
func (ch *Channel) ReadDataComplex64All(options ...ReadOption) ([]complex64, error) {
	return readAllData(ch, options, DataTypeComplex64, interpretComplex64)
}

// ReadDataComplex128All reads all complex128 values from the channel into a single slice.
func (ch *Channel) ReadDataComplex128All(options ...ReadOption) ([]complex128, error) {
	return readAllData(ch, options, DataTypeComplex128, interpretComplex128)
}
