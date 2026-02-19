package tdms

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// channelMetadata builds the metadata bytes for a standard (non-DAQmx) channel object
// with a normal raw data index header.
func channelMetadata(channelName string, dataType DataType, numValues uint64, props map[string]any) []byte {
	path := "/'" + defaultGroupName + "'/'" + channelName + "'"

	var buf bytes.Buffer

	// Path length and path
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(path)))
	buf.WriteString(path)

	// Raw data index header: 0x14 (20) means "normal raw data index follows"
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0x14))

	// Data type
	_ = binary.Write(&buf, binary.LittleEndian, uint32(dataType))

	// Array dimension (always 1 in TDMS v2)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(1))

	// Number of values
	_ = binary.Write(&buf, binary.LittleEndian, numValues)

	// For variable-length types (e.g. strings), we'd need to write totalSize
	// here too, but we handle that via a separate function.

	// Properties
	if props == nil {
		props = make(map[string]any)
	}
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(props)))
	for name, value := range props {
		writeProperty(&buf, name, value)
	}

	return buf.Bytes()
}

// channelMetadataWithTotalSize builds metadata for a variable-length channel (e.g. strings)
// that requires an explicit total size in the raw data index.
func channelMetadataWithTotalSize(channelName string, dataType DataType, numValues uint64, totalSize uint64, props map[string]any) []byte {
	path := "/'" + defaultGroupName + "'/'" + channelName + "'"

	var buf bytes.Buffer

	// Path length and path
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(path)))
	buf.WriteString(path)

	// Raw data index header
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0x14))

	// Data type
	_ = binary.Write(&buf, binary.LittleEndian, uint32(dataType))

	// Array dimension
	_ = binary.Write(&buf, binary.LittleEndian, uint32(1))

	// Number of values
	_ = binary.Write(&buf, binary.LittleEndian, numValues)

	// Total size (for variable-length types)
	_ = binary.Write(&buf, binary.LittleEndian, totalSize)

	// Properties
	if props == nil {
		props = make(map[string]any)
	}
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(props)))
	for name, value := range props {
		writeProperty(&buf, name, value)
	}

	return buf.Bytes()
}

// channelMetadataNoRawData builds metadata for an object that has no raw data
// (raw data index header = 0xFFFFFFFF).
func channelMetadataNoRawData(path string, props map[string]any) []byte {
	var buf bytes.Buffer

	// Path length and path
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(path)))
	buf.WriteString(path)

	// Raw data index: no raw data
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0xFFFFFFFF))

	// Properties
	if props == nil {
		props = make(map[string]any)
	}
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(props)))
	for name, value := range props {
		writeProperty(&buf, name, value)
	}

	return buf.Bytes()
}

// channelMetadataMatchesPrevious builds metadata for an object whose raw data
// index matches the previous segment (header = 0x00000000).
func channelMetadataMatchesPrevious(channelName string, props map[string]any) []byte {
	path := "/'" + defaultGroupName + "'/'" + channelName + "'"

	var buf bytes.Buffer

	// Path length and path
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(path)))
	buf.WriteString(path)

	// Raw data index: matches previous
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0x00000000))

	// Properties
	if props == nil {
		props = make(map[string]any)
	}
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(props)))
	for name, value := range props {
		writeProperty(&buf, name, value)
	}

	return buf.Bytes()
}

// standardSegmentTOC returns a TOC bitmask for a standard (non-DAQmx) segment
// with metadata, raw data, and a new object list.
func standardSegmentTOC() uint32 {
	return tocMetaData | tocRawData | tocNewObjList
}

// interleavedSegmentTOC returns a TOC for an interleaved segment.
func interleavedSegmentTOC() uint32 {
	return tocMetaData | tocRawData | tocNewObjList | (1 << 5)
}

const defaultGroupName = "Group"

// buildLeadIn constructs a 28-byte TDMS lead-in from its constituent parts.
func buildLeadIn(toc uint32, version uint32, nextSegmentOffset, rawDataOffset uint64) []byte {
	var buf bytes.Buffer

	buf.Write(tdmsMagicBytes)

	// TOC is always little endian
	_ = binary.Write(&buf, binary.LittleEndian, toc)

	// Version and offsets use the byte order indicated by the TOC
	order := binary.ByteOrder(binary.LittleEndian)
	if toc&(1<<6) != 0 {
		order = binary.BigEndian
	}

	_ = binary.Write(&buf, order, version)
	_ = binary.Write(&buf, order, nextSegmentOffset)
	_ = binary.Write(&buf, order, rawDataOffset)

	return buf.Bytes()
}

// buildStandardFile constructs a complete TDMS file with one segment containing
// standard (non-DAQmx) channels. Each entry in channelData maps channel name
// to raw data bytes.
func buildStandardFile(channels []testChannelDef) []byte {
	numObjects := uint32(2 + len(channels)) // root + group + channels

	objects := make([][]byte, 0, numObjects)
	objects = append(objects, rootMetadata())
	objects = append(objects, groupMetadata())

	var rawData bytes.Buffer
	for _, ch := range channels {
		objects = append(objects, ch.metadata)
		rawData.Write(ch.rawData)
	}

	metadata := buildMetadata(numObjects, objects...)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), metadata, rawData.Bytes())
	return tf.build()
}

// testChannelDef defines a channel for use with buildStandardFile.
type testChannelDef struct {
	metadata []byte
	rawData  []byte
}

// int16Channel creates a testChannelDef for an int16 channel.
func int16Channel(name string, values []int16) testChannelDef {
	var rawData bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeInt16, uint64(len(values)), nil),
		rawData:  rawData.Bytes(),
	}
}

// int32Channel creates a testChannelDef for an int32 channel.
func int32Channel(name string, values []int32) testChannelDef {
	var rawData bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeInt32, uint64(len(values)), nil),
		rawData:  rawData.Bytes(),
	}
}

// uint32Channel creates a testChannelDef for a uint32 channel.
func uint32Channel(name string, values []uint32) testChannelDef {
	var rawData bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeUint32, uint64(len(values)), nil),
		rawData:  rawData.Bytes(),
	}
}

// float32Channel creates a testChannelDef for a float32 channel.
func float32Channel(name string, values []float32) testChannelDef {
	var rawData bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeFloat32, uint64(len(values)), nil),
		rawData:  rawData.Bytes(),
	}
}

// float64Channel creates a testChannelDef for a float64 channel.
func float64Channel(name string, values []float64) testChannelDef {
	var rawData bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeFloat64, uint64(len(values)), nil),
		rawData:  rawData.Bytes(),
	}
}

// uint8Channel creates a testChannelDef for a uint8 channel.
func uint8Channel(name string, values []uint8) testChannelDef {
	rawData := make([]byte, len(values))
	copy(rawData, values)

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeUint8, uint64(len(values)), nil),
		rawData:  rawData,
	}
}

// boolChannel creates a testChannelDef for a bool channel.
func boolChannel(name string, values []bool) testChannelDef {
	rawData := make([]byte, len(values))
	for i, v := range values {
		if v {
			rawData[i] = 1
		}
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeBool, uint64(len(values)), nil),
		rawData:  rawData,
	}
}

// stringChannel creates a testChannelDef for a string channel.
// Strings in TDMS raw data format: first the offsets (uint32 per value), then the concatenated string bytes.
func stringChannel(name string, values []string) testChannelDef {
	var offsetBuf bytes.Buffer
	var dataBuf bytes.Buffer

	runningOffset := uint32(0)
	for _, s := range values {
		runningOffset += uint32(len(s))
		_ = binary.Write(&offsetBuf, binary.LittleEndian, runningOffset)
		dataBuf.WriteString(s)
	}

	totalSize := uint64(offsetBuf.Len() + dataBuf.Len())

	var rawData bytes.Buffer
	rawData.Write(offsetBuf.Bytes())
	rawData.Write(dataBuf.Bytes())

	return testChannelDef{
		metadata: channelMetadataWithTotalSize(name, DataTypeString, uint64(len(values)), totalSize, nil),
		rawData:  rawData.Bytes(),
	}
}

// int16ChannelWithProps creates a testChannelDef for an int16 channel with properties.
func int16ChannelWithProps(name string, values []int16, props map[string]any) testChannelDef {
	var rawData bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	return testChannelDef{
		metadata: channelMetadata(name, DataTypeInt16, uint64(len(values)), props),
		rawData:  rawData.Bytes(),
	}
}

// float64Bytes converts a float64 slice to its little-endian binary representation.
func float64Bytes(values ...float64) []byte {
	var buf bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// int32Bytes converts an int32 slice to its little-endian binary representation.
func int32Bytes(values ...int32) []byte {
	var buf bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// int16Bytes converts an int16 slice to its little-endian binary representation.
func int16Bytes(values ...int16) []byte {
	var buf bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// groupMetadataWithProps builds group metadata with properties.
func groupMetadataWithProps(props map[string]any) []byte {
	return channelMetadataNoRawData("/'"+defaultGroupName+"'", props)
}

// rootMetadataWithProps builds root metadata with properties.
func rootMetadataWithProps(props map[string]any) []byte {
	return channelMetadataNoRawData("/", props)
}

// requireNoError is a test helper that fails the test immediately if err is not nil.
func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireChannel is a test helper that gets a channel or fails.
func requireChannel(t *testing.T, file *File, groupName, channelName string) *Channel {
	t.Helper()
	ch, err := getChannel(file, groupName, channelName)
	if err != nil {
		t.Fatalf("failed to get channel %s/%s: %v", groupName, channelName, err)
	}
	return ch
}

// almostEqual checks whether two float64 values are within epsilon of each other.
func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}
