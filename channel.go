package tdms

import (
	"iter"
	"time"
)

// Channel represents a data channel within a [Group]. Use the ReadData methods
// to access the channel's data in a type-safe manner.
type Channel struct {
	Name       string
	GroupName  string
	DataType   DataType
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

func (ch *Channel) ReadDataAsInt8(options ...ReadOption) iter.Seq2[int8, error] {
	return StreamReader(ch, options, DataTypeInt8, interpretInt8)
}

func (ch *Channel) ReadDataAsInt16(options ...ReadOption) iter.Seq2[int16, error] {
	return StreamReader(ch, options, DataTypeInt16, interpretInt16)
}

func (ch *Channel) ReadDataAsInt32(options ...ReadOption) iter.Seq2[int32, error] {
	return StreamReader(ch, options, DataTypeInt32, interpretInt32)
}

func (ch *Channel) ReadDataAsInt64(options ...ReadOption) iter.Seq2[int64, error] {
	return StreamReader(ch, options, DataTypeInt64, interpretInt64)
}

func (ch *Channel) ReadDataAsUint8(options ...ReadOption) iter.Seq2[uint8, error] {
	return StreamReader(ch, options, DataTypeUint8, interpretUint8)
}

func (ch *Channel) ReadDataAsUint16(options ...ReadOption) iter.Seq2[uint16, error] {
	return StreamReader(ch, options, DataTypeUint16, interpretUint16)
}

func (ch *Channel) ReadDataAsUint32(options ...ReadOption) iter.Seq2[uint32, error] {
	return StreamReader(ch, options, DataTypeUint32, interpretUint32)
}

func (ch *Channel) ReadDataAsUint64(options ...ReadOption) iter.Seq2[uint64, error] {
	return StreamReader(ch, options, DataTypeUint64, interpretUint64)
}

func (ch *Channel) ReadDataAsFloat32(options ...ReadOption) iter.Seq2[float32, error] {
	return StreamReader(ch, options, DataTypeFloat32, interpretFloat32)
}

func (ch *Channel) ReadDataAsFloat64(options ...ReadOption) iter.Seq2[float64, error] {
	return StreamReader(ch, options, DataTypeFloat64, interpretFloat64)
}

func (ch *Channel) ReadDataAsFloat128(options ...ReadOption) iter.Seq2[Float128, error] {
	return StreamReader(ch, options, DataTypeFloat128, interpretFloat128)
}

func (ch *Channel) ReadDataAsString(options ...ReadOption) iter.Seq2[string, error] {
	return StreamReader(ch, options, DataTypeString, interpretString)
}

func (ch *Channel) ReadDataAsBool(options ...ReadOption) iter.Seq2[bool, error] {
	return StreamReader(ch, options, DataTypeBool, interpretBool)
}

func (ch *Channel) ReadDataAsTimestamp(options ...ReadOption) iter.Seq2[Timestamp, error] {
	return StreamReader(ch, options, DataTypeTimestamp, interpretTimestamp)
}

func (ch *Channel) ReadDataAsTime(options ...ReadOption) iter.Seq2[time.Time, error] {
	return StreamReader(ch, options, DataTypeTimestamp, interpretTime)
}

func (ch *Channel) ReadDataAsComplex64(options ...ReadOption) iter.Seq2[complex64, error] {
	return StreamReader(ch, options, DataTypeComplex64, interpretComplex64)
}

func (ch *Channel) ReadDataAsComplex128(options ...ReadOption) iter.Seq2[complex128, error] {
	return StreamReader(ch, options, DataTypeComplex128, interpretComplex128)
}

// Data streaming functions that yield items in batches.

func (ch *Channel) ReadDataAsInt8Batch(options ...ReadOption) iter.Seq2[[]int8, error] {
	return BatchStreamReader(ch, options, DataTypeInt8, interpretInt8)
}

func (ch *Channel) ReadDataAsInt16Batch(options ...ReadOption) iter.Seq2[[]int16, error] {
	return BatchStreamReader(ch, options, DataTypeInt16, interpretInt16)
}

func (ch *Channel) ReadDataAsInt32Batch(options ...ReadOption) iter.Seq2[[]int32, error] {
	return BatchStreamReader(ch, options, DataTypeInt32, interpretInt32)
}

func (ch *Channel) ReadDataAsInt64Batch(options ...ReadOption) iter.Seq2[[]int64, error] {
	return BatchStreamReader(ch, options, DataTypeInt64, interpretInt64)
}

func (ch *Channel) ReadDataAsUint8Batch(options ...ReadOption) iter.Seq2[[]uint8, error] {
	return BatchStreamReader(ch, options, DataTypeUint8, interpretUint8)
}

func (ch *Channel) ReadDataAsUint16Batch(options ...ReadOption) iter.Seq2[[]uint16, error] {
	return BatchStreamReader(ch, options, DataTypeUint16, interpretUint16)
}

func (ch *Channel) ReadDataAsUint32Batch(options ...ReadOption) iter.Seq2[[]uint32, error] {
	return BatchStreamReader(ch, options, DataTypeUint32, interpretUint32)
}

func (ch *Channel) ReadDataAsUint64Batch(options ...ReadOption) iter.Seq2[[]uint64, error] {
	return BatchStreamReader(ch, options, DataTypeUint64, interpretUint64)
}

func (ch *Channel) ReadDataAsFloat32Batch(options ...ReadOption) iter.Seq2[[]float32, error] {
	return BatchStreamReader(ch, options, DataTypeFloat32, interpretFloat32)
}

func (ch *Channel) ReadDataAsFloat64Batch(options ...ReadOption) iter.Seq2[[]float64, error] {
	return BatchStreamReader(ch, options, DataTypeFloat64, interpretFloat64)
}

func (ch *Channel) ReadDataAsFloat128Batch(options ...ReadOption) iter.Seq2[[]Float128, error] {
	return BatchStreamReader(ch, options, DataTypeFloat128, interpretFloat128)
}

func (ch *Channel) ReadDataAsStringBatch(options ...ReadOption) iter.Seq2[[]string, error] {
	return BatchStreamReader(ch, options, DataTypeString, interpretString)
}

func (ch *Channel) ReadDataAsBoolBatch(options ...ReadOption) iter.Seq2[[]bool, error] {
	return BatchStreamReader(ch, options, DataTypeBool, interpretBool)
}

func (ch *Channel) ReadDataAsTimestampBatch(options ...ReadOption) iter.Seq2[[]Timestamp, error] {
	return BatchStreamReader(ch, options, DataTypeTimestamp, interpretTimestamp)
}

func (ch *Channel) ReadDataAsTimeBatch(options ...ReadOption) iter.Seq2[[]time.Time, error] {
	return BatchStreamReader(ch, options, DataTypeTimestamp, interpretTime)
}

func (ch *Channel) ReadDataAsComplex64Batch(options ...ReadOption) iter.Seq2[[]complex64, error] {
	return BatchStreamReader(ch, options, DataTypeComplex64, interpretComplex64)
}

func (ch *Channel) ReadDataAsComplex128Batch(options ...ReadOption) iter.Seq2[[]complex128, error] {
	return BatchStreamReader(ch, options, DataTypeComplex128, interpretComplex128)
}

// Data streaming functions that read all the data for a channel in one go.

func (ch *Channel) ReadDataInt8All(options ...ReadOption) ([]int8, error) {
	return readAllData(ch, options, DataTypeInt8, interpretInt8)
}

func (ch *Channel) ReadDataInt16All(options ...ReadOption) ([]int16, error) {
	return readAllData(ch, options, DataTypeInt16, interpretInt16)
}

func (ch *Channel) ReadDataInt32All(options ...ReadOption) ([]int32, error) {
	return readAllData(ch, options, DataTypeInt32, interpretInt32)
}

func (ch *Channel) ReadDataInt64All(options ...ReadOption) ([]int64, error) {
	return readAllData(ch, options, DataTypeInt64, interpretInt64)
}

func (ch *Channel) ReadDataUint8All(options ...ReadOption) ([]uint8, error) {
	return readAllData(ch, options, DataTypeUint8, interpretUint8)
}

func (ch *Channel) ReadDataUint16All(options ...ReadOption) ([]uint16, error) {
	return readAllData(ch, options, DataTypeUint16, interpretUint16)
}

func (ch *Channel) ReadDataUint32All(options ...ReadOption) ([]uint32, error) {
	return readAllData(ch, options, DataTypeUint32, interpretUint32)
}

func (ch *Channel) ReadDataUint64All(options ...ReadOption) ([]uint64, error) {
	return readAllData(ch, options, DataTypeUint64, interpretUint64)
}

func (ch *Channel) ReadDataFloat32All(options ...ReadOption) ([]float32, error) {
	return readAllData(ch, options, DataTypeFloat32, interpretFloat32)
}

func (ch *Channel) ReadDataFloat64All(options ...ReadOption) ([]float64, error) {
	return readAllData(ch, options, DataTypeFloat64, interpretFloat64)
}

func (ch *Channel) ReadDataFloat128All(options ...ReadOption) ([]Float128, error) {
	return readAllData(ch, options, DataTypeFloat128, interpretFloat128)
}

func (ch *Channel) ReadDataStringAll(options ...ReadOption) ([]string, error) {
	return readAllData(ch, options, DataTypeString, interpretString)
}

func (ch *Channel) ReadDataBoolAll(options ...ReadOption) ([]bool, error) {
	return readAllData(ch, options, DataTypeBool, interpretBool)
}

func (ch *Channel) ReadDataTimestampAll(options ...ReadOption) ([]Timestamp, error) {
	return readAllData(ch, options, DataTypeTimestamp, interpretTimestamp)
}

func (ch *Channel) ReadDataTimeAll(options ...ReadOption) ([]time.Time, error) {
	return readAllData(ch, options, DataTypeTimestamp, interpretTime)
}

func (ch *Channel) ReadDataComplex64All(options ...ReadOption) ([]complex64, error) {
	return readAllData(ch, options, DataTypeComplex64, interpretComplex64)
}

func (ch *Channel) ReadDataComplex128All(options ...ReadOption) ([]complex128, error) {
	return readAllData(ch, options, DataTypeComplex128, interpretComplex128)
}
