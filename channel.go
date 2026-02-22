package tdms

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
	"time"
)

// Channel represents a data channel within a [Group]. Use the Read methods
// to access the channel's data in a type-safe manner.
type Channel struct {
	// Name is the name of this channel.
	Name string

	// GroupName is the name of the group that contains this channel.
	GroupName string

	// RawDataType is the type of raw data stored in this channel on disk.
	//
	// This is not necessarily the same as the data type of this channel once
	// scaling has been applied.
	RawDataType DataType

	// OutputDataType is the type of data in this channel once scaling has been applied.
	//
	// If data is read from this channel with scaling enabled (default), this
	// will be the type of the output scaling data. If scaling is disabled
	// during read using [WithScaling] option, the read method will return
	// data of type [RawDataType] instead.
	ScaledDataType DataType

	// Properties contains all properties associated with this channel.
	Properties Properties

	reader         io.ReadSeeker
	path           string
	dataChunks     []dataChunk
	totalNumValues uint64
	file           *File
	scaler         *Multiscaler
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
	offset            int64
	isInterleaved     bool
	isDAQmx           bool
	order             binary.ByteOrder
	size              uint64
	numValues         uint64
	stride            int64
	daqMXBufferWidths []uint32
	daqMXBufferSizes  []uint64
	daqMXScalers      map[int]DAQmxScaler
}

// NumValues returns the total number of data values in this channel across all
// segments.
func (ch *Channel) NumValues() uint64 {
	return ch.totalNumValues
}

func (ch *Channel) Unit() string {
	unit, _ := ch.Properties.GetString("unit_string")
	return unit
}

func (ch *Channel) Waveform() (Waveform, error) {
	xaxisName, _ := ch.Properties.GetString("wf_xname")

	xaxisUnit, err := ch.Properties.GetString("wf_xunit_string")
	if err != nil {
		return Waveform{}, fmt.Errorf("error getting waveform x-axis unit: %w", err)
	}

	numSamples, err := ch.Properties.GetUint("wf_samples")
	if err != nil {
		return Waveform{}, fmt.Errorf("error getting waveform number of samples: %w", err)
	}

	startOffset, err := ch.Properties.GetFloat("wf_start_offset")
	if err != nil {
		return Waveform{}, fmt.Errorf("error getting waveform start offset: %w", err)
	}

	increment, err := ch.Properties.GetFloat("wf_increment")
	if err != nil {
		return Waveform{}, fmt.Errorf("error getting waveform increment: %w", err)
	}

	var startTime *time.Time
	startTimeVal, err := ch.Properties.GetTimestamp("wf_start_time")
	if errors.Is(err, ErrPropertyNotFound) {
		// This property is optional – that's fine.
	} else if err != nil {
		return Waveform{}, fmt.Errorf("error getting waveform start time: %w", err)
	} else {
		startTime = ptr(startTimeVal.AsTime())
	}

	return Waveform{
		XAxisName:      xaxisName,
		XAxisUnit:      xaxisUnit,
		NumSamples:     uint(numSamples),
		StartOffset:    startOffset,
		StartTime:      startTime,
		TimePreference: TimePreferenceNone,
		increment:      increment,
	}, nil
}

type readOptions struct {
	batchSize       int
	shouldScale     bool
	daqmxScaleIndex int
}

func renderReadOptions(options []ReadOption) readOptions {
	opts := readOptions{
		batchSize:       0,
		shouldScale:     true,
		daqmxScaleIndex: 0,
	}

	for _, opt := range options {
		opt(&opts)
	}

	return opts
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
//
// DAQmx scalers are required to understand the raw data and do not perform
// mathematical transformations in the way that the other scalers do, so
// disabling scaling when reading DAQmx data has no effect.
func WithScaling(shouldScale ...bool) ReadOption {
	return func(opts *readOptions) {
		if len(shouldScale) > 0 {
			opts.shouldScale = shouldScale[0]
		} else {
			opts.shouldScale = true
		}
	}
}

func ForDAQmxScaler(daqmxScaleIndex int) ReadOption {
	return func(opts *readOptions) {
		opts.daqmxScaleIndex = daqmxScaleIndex
	}
}

func (ch *Channel) DataType(options ...ReadOption) (DataType, error) {
	opts := renderReadOptions(options)

	dataType := ch.RawDataType

	if opts.shouldScale {
		dataType = ch.ScaledDataType
	}

	// For DAQmx raw data with scaling disabled, use the DAQmx scaler's data type
	if dataType == DataTypeDAQmxRawData {
		if opts.daqmxScaleIndex >= len(ch.scaler.scalers) {
			return DataTypeVoid, fmt.Errorf("invalid DAQmx scale index: %d", opts.daqmxScaleIndex)
		}

		daqmxScaler, ok := ch.scaler.scalers[opts.daqmxScaleIndex].(*DAQmxScaler)
		if !ok {
			return DataTypeVoid, fmt.Errorf("expected DAQmx scaler, got %T", ch.scaler.scalers[opts.daqmxScaleIndex])
		}

		var err error
		dataType, err = daqmxScaler.OutputType(nil)
		if err != nil {
			return DataTypeVoid, fmt.Errorf("failed to get DAQmx scaler output type: %w", err)
		}
	}

	return dataType, nil
}

// Data streaming functions that yield each item at a time.

func (ch *Channel) Read(options ...ReadOption) iter.Seq2[any, error] {
	return convertStream[any](ch, options)
}

// ReadInt8 returns an iterator that yields individual int8 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt8(options ...ReadOption) iter.Seq2[int8, error] {
	return convertStream[int8](ch, options)
}

// ReadInt16 returns an iterator that yields individual int16 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt16(options ...ReadOption) iter.Seq2[int16, error] {
	return convertStream[int16](ch, options)
}

// ReadInt32 returns an iterator that yields individual int32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt32(options ...ReadOption) iter.Seq2[int32, error] {
	return convertStream[int32](ch, options)
}

// ReadInt64 returns an iterator that yields individual int64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadInt64(options ...ReadOption) iter.Seq2[int64, error] {
	return convertStream[int64](ch, options)
}

// ReadUint8 returns an iterator that yields individual uint8 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint8(options ...ReadOption) iter.Seq2[uint8, error] {
	return convertStream[uint8](ch, options)
}

// ReadUint16 returns an iterator that yields individual uint16 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint16(options ...ReadOption) iter.Seq2[uint16, error] {
	return convertStream[uint16](ch, options)
}

// ReadUint32 returns an iterator that yields individual uint32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint32(options ...ReadOption) iter.Seq2[uint32, error] {
	return convertStream[uint32](ch, options)
}

// ReadUint64 returns an iterator that yields individual uint64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadUint64(options ...ReadOption) iter.Seq2[uint64, error] {
	return convertStream[uint64](ch, options)
}

// ReadFloat32 returns an iterator that yields individual float32 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadFloat32(options ...ReadOption) iter.Seq2[float32, error] {
	return convertStream[float32](ch, options)
}

// ReadFloat64 returns an iterator that yields individual float64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadFloat64(options ...ReadOption) iter.Seq2[float64, error] {
	return convertStream[float64](ch, options)
}

// ReadFloat128 returns an iterator that yields individual [Float128] values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadFloat128(options ...ReadOption) iter.Seq2[Float128, error] {
	return convertStream[Float128](ch, options)
}

// ReadString returns an iterator that yields individual string values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadString(options ...ReadOption) iter.Seq2[string, error] {
	return convertStream[string](ch, options)
}

// ReadBool returns an iterator that yields individual bool values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadBool(options ...ReadOption) iter.Seq2[bool, error] {
	return convertStream[bool](ch, options)
}

// ReadTimestamp returns an iterator that yields individual [Timestamp] values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadTimestamp(options ...ReadOption) iter.Seq2[Timestamp, error] {
	return convertStream[Timestamp](ch, options)
}

// ReadComplex64 returns an iterator that yields individual complex64 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadComplex64(options ...ReadOption) iter.Seq2[complex64, error] {
	return convertStream[complex64](ch, options)
}

// ReadComplex128 returns an iterator that yields individual complex128 values from the channel.
// Use BatchSize option to control internal buffer size.
func (ch *Channel) ReadComplex128(options ...ReadOption) iter.Seq2[complex128, error] {
	return convertStream[complex128](ch, options)
}

// Data streaming functions that yield items in batches.

// ReadBatch returns an iterator that yields batches of values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadBatch(options ...ReadOption) iter.Seq2[any, error] {
	// This one is a bit different because we don't want to return []any, we
	// want to return an any that can be []int8, []int16, etc.
	dataType, err := ch.DataType(options...)
	if err != nil {
		return func(yield func(any, error) bool) {
			yield(nil, fmt.Errorf("failed to get channel data type: %w", err))
		}
	}

	switch dataType {
	case DataTypeInt8:
		return toAnySeq2(convertBatchStream[int8](ch, options))
	case DataTypeInt16:
		return toAnySeq2(convertBatchStream[int16](ch, options))
	case DataTypeInt32:
		return toAnySeq2(convertBatchStream[int32](ch, options))
	case DataTypeInt64:
		return toAnySeq2(convertBatchStream[int64](ch, options))
	case DataTypeUint8:
		return toAnySeq2(convertBatchStream[uint8](ch, options))
	case DataTypeUint16:
		return toAnySeq2(convertBatchStream[uint16](ch, options))
	case DataTypeUint32:
		return toAnySeq2(convertBatchStream[uint32](ch, options))
	case DataTypeUint64:
		return toAnySeq2(convertBatchStream[uint64](ch, options))
	case DataTypeFloat32:
		return toAnySeq2(convertBatchStream[float32](ch, options))
	case DataTypeFloat64:
		return toAnySeq2(convertBatchStream[float64](ch, options))
	case DataTypeFloat128:
		return toAnySeq2(convertBatchStream[Float128](ch, options))
	case DataTypeBool:
		return toAnySeq2(convertBatchStream[bool](ch, options))
	case DataTypeString:
		return toAnySeq2(convertBatchStream[string](ch, options))
	case DataTypeTimestamp:
		return toAnySeq2(convertBatchStream[Timestamp](ch, options))
	case DataTypeComplex64:
		return toAnySeq2(convertBatchStream[complex64](ch, options))
	case DataTypeComplex128:
		return toAnySeq2(convertBatchStream[complex128](ch, options))
	default:
		panic(fmt.Sprintf("unsupported output data type for channel: %s", dataType))
	}
}

// ReadInt8Batch returns an iterator that yields batches of int8 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt8Batch(options ...ReadOption) iter.Seq2[[]int8, error] {
	return convertBatchStream[int8](ch, options)
}

// ReadInt16Batch returns an iterator that yields batches of int16 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt16Batch(options ...ReadOption) iter.Seq2[[]int16, error] {
	return convertBatchStream[int16](ch, options)
}

// ReadInt32Batch returns an iterator that yields batches of int32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt32Batch(options ...ReadOption) iter.Seq2[[]int32, error] {
	return convertBatchStream[int32](ch, options)
}

// ReadInt64Batch returns an iterator that yields batches of int64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadInt64Batch(options ...ReadOption) iter.Seq2[[]int64, error] {
	return convertBatchStream[int64](ch, options)
}

// ReadUint8Batch returns an iterator that yields batches of uint8 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint8Batch(options ...ReadOption) iter.Seq2[[]uint8, error] {
	return convertBatchStream[uint8](ch, options)
}

// ReadUint16Batch returns an iterator that yields batches of uint16 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint16Batch(options ...ReadOption) iter.Seq2[[]uint16, error] {
	return convertBatchStream[uint16](ch, options)
}

// ReadUint32Batch returns an iterator that yields batches of uint32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint32Batch(options ...ReadOption) iter.Seq2[[]uint32, error] {
	return convertBatchStream[uint32](ch, options)
}

// ReadUint64Batch returns an iterator that yields batches of uint64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadUint64Batch(options ...ReadOption) iter.Seq2[[]uint64, error] {
	return convertBatchStream[uint64](ch, options)
}

// ReadFloat32Batch returns an iterator that yields batches of float32 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadFloat32Batch(options ...ReadOption) iter.Seq2[[]float32, error] {
	return convertBatchStream[float32](ch, options)
}

// ReadFloat64Batch returns an iterator that yields batches of float64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadFloat64Batch(options ...ReadOption) iter.Seq2[[]float64, error] {
	return convertBatchStream[float64](ch, options)
}

// ReadFloat128Batch returns an iterator that yields batches of [Float128] values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadFloat128Batch(options ...ReadOption) iter.Seq2[[]Float128, error] {
	return convertBatchStream[Float128](ch, options)
}

// ReadStringBatch returns an iterator that yields batches of string values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadStringBatch(options ...ReadOption) iter.Seq2[[]string, error] {
	return convertBatchStream[string](ch, options)
}

// ReadBoolBatch returns an iterator that yields batches of bool values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadBoolBatch(options ...ReadOption) iter.Seq2[[]bool, error] {
	return convertBatchStream[bool](ch, options)
}

// ReadTimestampBatch returns an iterator that yields batches of [Timestamp] values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadTimestampBatch(options ...ReadOption) iter.Seq2[[]Timestamp, error] {
	return convertBatchStream[Timestamp](ch, options)
}

// ReadComplex64Batch returns an iterator that yields batches of complex64 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadComplex64Batch(options ...ReadOption) iter.Seq2[[]complex64, error] {
	return convertBatchStream[complex64](ch, options)
}

// ReadComplex128Batch returns an iterator that yields batches of complex128 values from the channel.
// Use BatchSize option to control batch size.
func (ch *Channel) ReadComplex128Batch(options ...ReadOption) iter.Seq2[[]complex128, error] {
	return convertBatchStream[complex128](ch, options)
}

// Data streaming functions that read all the data for a channel in one go.

func (ch *Channel) ReadAll(options ...ReadOption) (any, error) {
	dataType, err := ch.DataType(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel data type: %w", err)
	}

	switch dataType {
	case DataTypeInt8:
		return readAllData[int8](ch, options)
	case DataTypeInt16:
		return readAllData[int16](ch, options)
	case DataTypeInt32:
		return readAllData[int32](ch, options)
	case DataTypeInt64:
		return readAllData[int64](ch, options)
	case DataTypeUint8:
		return readAllData[uint8](ch, options)
	case DataTypeUint16:
		return readAllData[uint16](ch, options)
	case DataTypeUint32:
		return readAllData[uint32](ch, options)
	case DataTypeUint64:
		return readAllData[uint64](ch, options)
	case DataTypeFloat32:
		return readAllData[float32](ch, options)
	case DataTypeFloat64:
		return readAllData[float64](ch, options)
	case DataTypeString:
		return readAllData[string](ch, options)
	case DataTypeBool:
		return readAllData[bool](ch, options)
	case DataTypeTimestamp:
		return readAllData[Timestamp](ch, options)
	case DataTypeComplex64:
		return readAllData[complex64](ch, options)
	case DataTypeComplex128:
		return readAllData[complex128](ch, options)
	default:
		panic(fmt.Sprintf("unsupported output data type for channel: %s", dataType))
	}
}

// ReadInt8All reads all int8 values from the channel into a single slice.
func (ch *Channel) ReadInt8All(options ...ReadOption) ([]int8, error) {
	return readAllData[int8](ch, options)
}

// ReadInt16All reads all int16 values from the channel into a single slice.
func (ch *Channel) ReadInt16All(options ...ReadOption) ([]int16, error) {
	return readAllData[int16](ch, options)
}

// ReadInt32All reads all int32 values from the channel into a single slice.
func (ch *Channel) ReadInt32All(options ...ReadOption) ([]int32, error) {
	return readAllData[int32](ch, options)
}

// ReadInt64All reads all int64 values from the channel into a single slice.
func (ch *Channel) ReadInt64All(options ...ReadOption) ([]int64, error) {
	return readAllData[int64](ch, options)
}

// ReadUint8All reads all uint8 values from the channel into a single slice.
func (ch *Channel) ReadUint8All(options ...ReadOption) ([]uint8, error) {
	return readAllData[uint8](ch, options)
}

// ReadUint16All reads all uint16 values from the channel into a single slice.
func (ch *Channel) ReadUint16All(options ...ReadOption) ([]uint16, error) {
	return readAllData[uint16](ch, options)
}

// ReadUint32All reads all uint32 values from the channel into a single slice.
func (ch *Channel) ReadUint32All(options ...ReadOption) ([]uint32, error) {
	return readAllData[uint32](ch, options)
}

// ReadUint64All reads all uint64 values from the channel into a single slice.
func (ch *Channel) ReadUint64All(options ...ReadOption) ([]uint64, error) {
	return readAllData[uint64](ch, options)
}

// ReadFloat32All reads all float32 values from the channel into a single slice.
func (ch *Channel) ReadFloat32All(options ...ReadOption) ([]float32, error) {
	return readAllData[float32](ch, options)
}

// ReadFloat64All reads all float64 values from the channel into a single slice.
func (ch *Channel) ReadFloat64All(options ...ReadOption) ([]float64, error) {
	return readAllData[float64](ch, options)
}

// ReadFloat128All reads all [Float128] values from the channel into a single slice.
func (ch *Channel) ReadFloat128All(options ...ReadOption) ([]Float128, error) {
	return readAllData[Float128](ch, options)
}

// ReadStringAll reads all string values from the channel into a single slice.
func (ch *Channel) ReadStringAll(options ...ReadOption) ([]string, error) {
	return readAllData[string](ch, options)
}

// ReadBoolAll reads all bool values from the channel into a single slice.
func (ch *Channel) ReadBoolAll(options ...ReadOption) ([]bool, error) {
	return readAllData[bool](ch, options)
}

// ReadTimestampAll reads all [Timestamp] values from the channel into a single slice.
func (ch *Channel) ReadTimestampAll(options ...ReadOption) ([]Timestamp, error) {
	return readAllData[Timestamp](ch, options)
}

// ReadComplex64All reads all complex64 values from the channel into a single slice.
func (ch *Channel) ReadComplex64All(options ...ReadOption) ([]complex64, error) {
	return readAllData[complex64](ch, options)
}

// ReadComplex128All reads all complex128 values from the channel into a single slice.
func (ch *Channel) ReadComplex128All(options ...ReadOption) ([]complex128, error) {
	return readAllData[complex128](ch, options)
}

func convertStream[T any](ch *Channel, options []ReadOption) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for batch, err := range convertBatchStream[T](ch, options) {
			if err != nil {
				yield(*new(T), err)
				return
			}

			for _, v := range batch {
				if !yield(v, nil) {
					return
				}
			}
		}
	}
}

func convertBatchStream[T any](ch *Channel, options []ReadOption) iter.Seq2[[]T, error] {
	return func(yield func([]T, error) bool) {
		var streamer iter.Seq2[any, error]

		// The generic in the stream reader is the raw type, not the output type.
		switch ch.RawDataType {
		case DataTypeInt8:
			streamer = batchStreamReader[int8](ch, options)
		case DataTypeInt16:
			streamer = batchStreamReader[int16](ch, options)
		case DataTypeInt32:
			streamer = batchStreamReader[int32](ch, options)
		case DataTypeInt64:
			streamer = batchStreamReader[int64](ch, options)
		case DataTypeUint8:
			streamer = batchStreamReader[uint8](ch, options)
		case DataTypeUint16:
			streamer = batchStreamReader[uint16](ch, options)
		case DataTypeUint32:
			streamer = batchStreamReader[uint32](ch, options)
		case DataTypeUint64:
			streamer = batchStreamReader[uint64](ch, options)
		case DataTypeFloat32:
			streamer = batchStreamReader[float32](ch, options)
		case DataTypeFloat64:
			streamer = batchStreamReader[float64](ch, options)
		case DataTypeFloat128:
			streamer = batchStreamReader[Float128](ch, options)
		case DataTypeString:
			streamer = batchStreamReader[string](ch, options)
		case DataTypeBool:
			streamer = batchStreamReader[bool](ch, options)
		case DataTypeTimestamp:
			streamer = batchStreamReader[Timestamp](ch, options)
		case DataTypeComplex64:
			streamer = batchStreamReader[complex64](ch, options)
		case DataTypeComplex128:
			streamer = batchStreamReader[complex128](ch, options)
		default:
			yield(nil, ErrUnsupportedType)
			return
		}

		for batch, err := range streamer {
			v, ok := batch.([]T)
			if !ok {
				yield(nil, fmt.Errorf("failed to convert value to []%T", v))
				return
			}
			if !yield(v, err) {
				return
			}
		}
	}
}

func toAnySeq2[V any](seq iter.Seq2[V, error]) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		seq(func(v V, err error) bool {
			return yield(v, err)
		})
	}
}

// readAllData reads all data from a channel and put it into a single slice.
//
// By re-using BatchStreamReader here, we can avoid having to allocate 2*N bytes
// – one for the raw bytes and other for the interpreted values. The raw bytes
// are still batched while we allocate the values slice up-front. It's also
// cleaner in terms of the code as we avoid re-implementing the underlying read
// functionality.
func readAllData[T any](ch *Channel, options []ReadOption) ([]T, error) {
	// Remember, scaling can change the type so that it's not the same as what's
	// set in the channel metadata (i.e. T).
	var values any

	dataType, err := ch.DataType(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel data type: %w", err)
	}

	switch dataType {
	case DataTypeInt8:
		values = make([]int8, 0, ch.totalNumValues)
	case DataTypeInt16:
		values = make([]int16, 0, ch.totalNumValues)
	case DataTypeInt32:
		values = make([]int32, 0, ch.totalNumValues)
	case DataTypeInt64:
		values = make([]int64, 0, ch.totalNumValues)
	case DataTypeUint8:
		values = make([]uint8, 0, ch.totalNumValues)
	case DataTypeUint16:
		values = make([]uint16, 0, ch.totalNumValues)
	case DataTypeUint32:
		values = make([]uint32, 0, ch.totalNumValues)
	case DataTypeUint64:
		values = make([]uint64, 0, ch.totalNumValues)
	case DataTypeFloat32, DataTypeFloat32WithUnit:
		values = make([]float32, 0, ch.totalNumValues)
	case DataTypeFloat64, DataTypeFloat64WithUnit:
		values = make([]float64, 0, ch.totalNumValues)
	case DataTypeFloat128, DataTypeFloat128WithUnit:
		values = make([]Float128, 0, ch.totalNumValues)
	case DataTypeBool:
		values = make([]bool, 0, ch.totalNumValues)
	case DataTypeString:
		values = make([]string, 0, ch.totalNumValues)
	case DataTypeTimestamp:
		values = make([]Timestamp, 0, ch.totalNumValues)
	case DataTypeComplex64:
		values = make([]complex64, 0, ch.totalNumValues)
	case DataTypeComplex128:
		values = make([]complex128, 0, ch.totalNumValues)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dataType)
	}

	var streamer iter.Seq2[any, error]

	// The generic in the stream reader is the raw type, not the output type. We
	// can emulate this by getting the data type and passing in the option to
	// disable scaling. We need to put the scaling disable option at the end to
	// overwrite any user-input option that enables it.
	optionsNoScaling := make([]ReadOption, 0, len(options)+1)
	optionsNoScaling = append(optionsNoScaling, options...)
	optionsNoScaling = append(optionsNoScaling, WithScaling(false))

	rawDataType, err := ch.DataType(optionsNoScaling...)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel raw data type: %w", err)
	}

	switch rawDataType {
	case DataTypeInt8:
		streamer = batchStreamReader[int8](ch, options)
	case DataTypeInt16:
		streamer = batchStreamReader[int16](ch, options)
	case DataTypeInt32:
		streamer = batchStreamReader[int32](ch, options)
	case DataTypeInt64:
		streamer = batchStreamReader[int64](ch, options)
	case DataTypeUint8:
		streamer = batchStreamReader[uint8](ch, options)
	case DataTypeUint16:
		streamer = batchStreamReader[uint16](ch, options)
	case DataTypeUint32:
		streamer = batchStreamReader[uint32](ch, options)
	case DataTypeUint64:
		streamer = batchStreamReader[uint64](ch, options)
	case DataTypeFloat32:
		streamer = batchStreamReader[float32](ch, options)
	case DataTypeFloat64:
		streamer = batchStreamReader[float64](ch, options)
	case DataTypeFloat128:
		streamer = batchStreamReader[Float128](ch, options)
	case DataTypeString:
		streamer = batchStreamReader[string](ch, options)
	case DataTypeBool:
		streamer = batchStreamReader[bool](ch, options)
	case DataTypeTimestamp:
		streamer = batchStreamReader[Timestamp](ch, options)
	case DataTypeComplex64:
		streamer = batchStreamReader[complex64](ch, options)
	case DataTypeComplex128:
		streamer = batchStreamReader[complex128](ch, options)
	default:
		return nil, ErrUnsupportedType
	}

	for batch, err := range streamer {
		if err != nil {
			return nil, err
		}

		switch v := batch.(type) {
		case []int8:
			values = append(values.([]int8), v...)
		case []int16:
			values = append(values.([]int16), v...)
		case []int32:
			values = append(values.([]int32), v...)
		case []int64:
			values = append(values.([]int64), v...)
		case []uint8:
			values = append(values.([]uint8), v...)
		case []uint16:
			values = append(values.([]uint16), v...)
		case []uint32:
			values = append(values.([]uint32), v...)
		case []uint64:
			values = append(values.([]uint64), v...)
		case []float32:
			values = append(values.([]float32), v...)
		case []float64:
			values = append(values.([]float64), v...)
		case []Float128:
			values = append(values.([]Float128), v...)
		case []string:
			values = append(values.([]string), v...)
		case []bool:
			values = append(values.([]bool), v...)
		case []Timestamp:
			values = append(values.([]Timestamp), v...)
		case []complex64:
			values = append(values.([]complex64), v...)
		case []complex128:
			values = append(values.([]complex128), v...)
		default:
			return nil, ErrUnsupportedType
		}
	}

	return values.([]T), nil
}
