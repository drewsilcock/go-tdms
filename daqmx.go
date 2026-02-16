package tdms

// DAQmx data is a variable-width interleaved vector data format which contains
// multiple internal data streams, each of which can have one of these types.
//
// The documentation around DAQmx and TDMS is essentially non-existent. Even
// things like the type enumeration below are completely undocumented. As such,
// we largely rely on the behaviour of the npTDMS library to determine how to
// understand and interpret the DAQmx data within the TDMS files. The logic we
// use to parse out and read the data is not exactly the same, but we replicate
// the same underlying behaviour and understanding, e.g. the type enum. Any
// inaccuracies in the npTDMS behaviour will therefore be replicated here.
//
// DAQmx requires "scalers" – these are not really scalers at all, but rather
// data "interpreters" that tell you how to extract the underlying DAQmx data
// streams from the bytes in the raw data chunks. We generally prefer the
// terminology "stream" to "scaler" for this reason, although if you are used to
// the NI terminology, you can replace "DAQmx stream" with "DAQmx scaler".
//
// Functionality in the "scaling.go" file are for actual scalers, i.e.
// parameters that define some kind of mathematical transformation on the data
// stream to reproduce physically accurate values/units from the stored data,
// e.g. retrieving degrees celsius from values from a temperature probe.
//
// The TDMS still counts DAQmx scalers as normal scalers in terms of the
// properties, e.g. "NI_Number_Of_Scales", it just reads the scales from object
// metadata instead of properties.

// DAQmxDataType represents the data type of a DAQmx channel.
//
// This is different from the normal TDS [DataType]. The enumeration values are
// different and only a subset of the values are supported (notably missing
// complex types, fixed point types, string type and boolean type).
type DAQmxDataType uint32

const (
	DAQmxDataTypeUint8 DAQmxDataType = iota
	DAQmxDataTypeInt8
	DAQmxDataTypeUint16
	DAQmxDataTypeInt16
	DAQmxDataTypeUint32
	DAQmxDataTypeInt32
	DAQmxDataTypeUint64
	DAQmxDataTypeInt64
	DAQmxDataTypeFloat32
	DAQmxDataTypeFloat64
	DAQmxDataTypeTimestamp DAQmxDataType = 0xffffffff
)

type daqmxScalerType int

const (
	daqmxScalerTypeNone daqmxScalerType = iota
	daqmxScalerTypeFormatChanging
	daqmxScalerTypeDigitalLine
)

// Multiple DAQmx channels can read from the same "buffer", while a single
// channel can consist of multiple streams of data interleaved together via
// multiple scalers. As such, DAQmx data can be multiply nested, where data for
// multiple scalers can be interleaved inside the data for each channel, which
// are interleaved themselves among the many channels.
//
// Each buffer corresponds to a single acquisition card. Each buffer appears in
// the data in its entirety before the next buffer appears (i.e. buffers are not
// themselves interleaved).

type daqmxScaler struct {
	// Format changing scaler essentially means "no-op" while digital line
	// scaler means the values are stored inside specific bits of the underlying
	// DAQmx data stream bytes.
	scalerType daqmxScalerType

	// Importantly, DAQmx data type is different from standard TDMS data type,
	// in particular fewer types are supported.
	dataType DAQmxDataType

	// rawBufferIndex indicates which buffer the DAQmx data stream pulls its
	// data from.
	rawBufferIndex uint32

	// This is the offset into the
	// For format changing, this is bytes. For digital line, this is bits.
	offsetWithinStride uint32

	// No-one seems to know what this means or does, and the documentation is
	// silent on the matter. We store it as uint32 for consistency, although we
	// expect digital line scalers to store it as uint8 (although this is
	// undocumented).
	sampleFormatBitmap uint32

	// Each scaler has an ID which is used to identify it uniquely. They often
	// but not always start from 0 or 1 and go up by whole numbers, although
	// that's only a convention.
	scaleID uint32
}

type formatChangingScaler daqmxScaler

func (s *formatChangingScaler) ReadProperties(_props Properties, _scaleIndex int) error {
	// DAQmx scalers read data from object raw data index, not properties.
	return nil
}

func (s *formatChangingScaler) InputSources() []int {
	return nil
}

func (s *formatChangingScaler) OutputType(inputTypes []DataType) (DataType, error) {
	return DataTypeVoid, nil
}

func (s *formatChangingScaler) Scale(inputs []any, output any) error {
	// TODO: Figure out how to represent this.
	return nil
}

type digitalLineScaler daqmxScaler

func (s *digitalLineScaler) InputSources() []int {
	return nil
}

func (s *digitalLineScaler) OutputType(inputTypes []DataType) (DataType, error) {
	return DataTypeVoid, nil
}

func (s *digitalLineScaler) ReadProperties(_props Properties, _scaleIndex int) error {
	// DAQmx scalers read data from object raw data index, not properties.
	return nil
}

func (s *digitalLineScaler) Scale(inputs []any, output any) error {
	// TODO: Figure out how to represent this.
	return nil
}
