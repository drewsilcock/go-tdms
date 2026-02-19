package tdms

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Test utilities for generating TDMS files with DAQmx data

type testFile struct {
	segments []testSegment
}

type testSegment struct {
	toc      uint32
	metadata []byte
	rawData  []byte
}

func newTestFile() *testFile {
	return &testFile{}
}

func (f *testFile) addSegment(toc uint32, metadata []byte, rawData []byte) {
	f.segments = append(f.segments, testSegment{
		toc:      toc,
		metadata: metadata,
		rawData:  rawData,
	})
}

func (f *testFile) build() []byte {
	var buf bytes.Buffer

	for _, seg := range f.segments {
		// Lead in
		buf.Write(tdmsMagicBytes)

		// TOC
		binary.Write(&buf, binary.LittleEndian, seg.toc)

		// Version
		binary.Write(&buf, binary.LittleEndian, uint32(4712))

		// Next segment offset and raw data offset
		// nextSegmentOffset is the offset from the end of the lead-in to the next segment
		// For the last segment, it's the total size of this segment (metadata + raw data)
		nextOffset := uint64(len(seg.metadata) + len(seg.rawData))
		rawDataOffset := uint64(len(seg.metadata))

		binary.Write(&buf, binary.LittleEndian, nextOffset)
		binary.Write(&buf, binary.LittleEndian, rawDataOffset)

		// Metadata
		buf.Write(seg.metadata)

		// Raw data
		buf.Write(seg.rawData)
	}

	return buf.Bytes()
}

// TOC bit flags
const (
	tocMetaData     = 1 << 1
	tocRawData      = 1 << 3
	tocNewObjList   = 1 << 2
	tocDAQmxRawData = 1 << 7
)

func segmentTOC() uint32 {
	return tocMetaData | tocRawData | tocNewObjList | tocDAQmxRawData
}

func segmentTOCNonDAQmx() uint32 {
	return tocMetaData | tocRawData | tocNewObjList
}

// Metadata builders

func buildMetadata(numObjects uint32, objects ...[]byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, numObjects)
	for _, obj := range objects {
		buf.Write(obj)
	}
	return buf.Bytes()
}

func rootMetadata() []byte {
	return objectMetadata("/", 0x00000000, 0)
}

func groupMetadata() []byte {
	return objectMetadata("/'Group'", 0x00000000, 0)
}

func objectMetadata(path string, rawDataIndex uint32, numProps uint32) []byte {
	var buf bytes.Buffer

	// Path length and path
	binary.Write(&buf, binary.LittleEndian, uint32(len(path)))
	buf.WriteString(path)

	// Raw data index
	binary.Write(&buf, binary.LittleEndian, rawDataIndex)

	// Number of properties
	binary.Write(&buf, binary.LittleEndian, numProps)

	return buf.Bytes()
}

func daqmxChannelMetadata(channelName string, numValues uint64, rawDataWidths []uint32, scalers [][]byte, dataType uint32, props map[string]any) []byte {
	path := fmt.Sprintf("/'Group'/'%s'", channelName)

	var buf bytes.Buffer

	// Path length and path
	binary.Write(&buf, binary.LittleEndian, uint32(len(path)))
	buf.WriteString(path)

	// Raw data index (0x1269 for format changing scaler, 0x126A for digital line)
	isDigital := false
	if len(scalers) > 0 && len(scalers[0]) == 13 { // Digital line scalers are 13 bytes
		isDigital = true
	}
	if isDigital {
		binary.Write(&buf, binary.LittleEndian, uint32(0x126A))
	} else {
		binary.Write(&buf, binary.LittleEndian, uint32(0x1269))
	}

	// Data type
	binary.Write(&buf, binary.LittleEndian, dataType)

	// Array dimension
	binary.Write(&buf, binary.LittleEndian, uint32(1))

	// Number of values
	binary.Write(&buf, binary.LittleEndian, numValues)

	// Number of scalers
	binary.Write(&buf, binary.LittleEndian, uint32(len(scalers)))

	// Scaler metadata
	for _, scaler := range scalers {
		buf.Write(scaler)
	}

	// Number of raw data widths
	binary.Write(&buf, binary.LittleEndian, uint32(len(rawDataWidths)))

	// Raw data widths
	for _, width := range rawDataWidths {
		binary.Write(&buf, binary.LittleEndian, width)
	}

	// Properties
	numProps := uint32(len(props))
	binary.Write(&buf, binary.LittleEndian, numProps)
	for name, value := range props {
		writeProperty(&buf, name, value)
	}

	return buf.Bytes()
}

func writeProperty(buf *bytes.Buffer, name string, value any) {
	// Property name length and name
	binary.Write(buf, binary.LittleEndian, uint32(len(name)))
	buf.WriteString(name)

	// Property data type and value
	switch v := value.(type) {
	case uint32:
		binary.Write(buf, binary.LittleEndian, uint32(0x03)) // tdsTypeU32
		binary.Write(buf, binary.LittleEndian, v)
	case int32:
		binary.Write(buf, binary.LittleEndian, uint32(0x05)) // tdsTypeI32
		binary.Write(buf, binary.LittleEndian, v)
	default:
		panic(fmt.Sprintf("unsupported property type: %T", value))
	}
}

func daqmxScalerMetadata(scaleID, dataType, byteOffset, rawBufferIndex uint32) []byte {
	var buf bytes.Buffer

	// DAQmx data type
	binary.Write(&buf, binary.LittleEndian, dataType)

	// Raw buffer index
	binary.Write(&buf, binary.LittleEndian, rawBufferIndex)

	// Raw byte offset
	binary.Write(&buf, binary.LittleEndian, byteOffset)

	// Sample format bitmap (unknown purpose)
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// Scale ID
	binary.Write(&buf, binary.LittleEndian, scaleID)

	return buf.Bytes()
}

func digitalScalerMetadata(scaleID, dataType, bitOffset, rawBufferIndex uint32) []byte {
	var buf bytes.Buffer

	// DAQmx data type
	binary.Write(&buf, binary.LittleEndian, dataType)

	// Raw buffer index
	binary.Write(&buf, binary.LittleEndian, rawBufferIndex)

	// Raw bit offset
	binary.Write(&buf, binary.LittleEndian, bitOffset)

	// Sample format bitmap (1 byte for digital)
	buf.WriteByte(0)

	// Scale ID
	binary.Write(&buf, binary.LittleEndian, scaleID)

	return buf.Bytes()
}

// Helper to convert hex string to bytes
func hexToBytes(hexStr string) []byte {
	// Remove whitespace
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "\n", "")
	hexStr = strings.ReplaceAll(hexStr, "\t", "")

	result := make([]byte, len(hexStr)/2)
	for i := range result {
		fmt.Sscanf(hexStr[i*2:i*2+2], "%02x", &result[i])
	}
	return result
}

// Actual tests

func TestSingleChannelInt16(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 3, 0, 0) // scale_id=0, type=Int16, offset=0, buffer=0

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{2}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 02 00 FF FF FE FF")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	// Read all data with scaling disabled to get raw DAQmx data
	data, err := channel.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel data: %v", err)
	}

	// Should return []int16
	int16Data, ok := data.([]int16)
	if !ok {
		t.Fatalf("Expected data type []int16, got %T", data)
	}

	expected := []int16{1, 2, -1, -2}
	if !cmp.Equal(int16Data, expected) {
		t.Errorf("Data mismatch:\n%s", cmp.Diff(expected, int16Data))
	}
}

func TestSingleChannelUint16(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 2, 0, 0) // type=Uint16

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{2}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 02 00 FF FF FE FF")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	data, err := channel.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel data: %v", err)
	}

	uint16Data, ok := data.([]uint16)
	if !ok {
		t.Fatalf("Expected data type []uint16, got %T", data)
	}

	expected := []uint16{1, 2, 65535, 65534}
	if !cmp.Equal(uint16Data, expected) {
		t.Errorf("Data mismatch:\n%s", cmp.Diff(expected, uint16Data))
	}
}

func TestSingleChannelInt32(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 5, 0, 0) // type=Int32

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{4}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 00 00 02 00 00 00 FF FF FF FF FE FF FF FF")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	data, err := channel.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel data: %v", err)
	}

	int32Data, ok := data.([]int32)
	if !ok {
		t.Fatalf("Expected data type []int32, got %T", data)
	}

	expected := []int32{1, 2, -1, -2}
	if !cmp.Equal(int32Data, expected) {
		t.Errorf("Data mismatch:\n%s", cmp.Diff(expected, int32Data))
	}
}

func TestSingleChannelUint32(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 4, 0, 0) // type=Uint32

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{4}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 00 00 02 00 00 00 FF FF FF FF FE FF FF FF")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	data, err := channel.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel data: %v", err)
	}

	uint32Data, ok := data.([]uint32)
	if !ok {
		t.Fatalf("Expected data type []uint32, got %T", data)
	}

	expected := []uint32{1, 2, 4294967295, 4294967294}
	if !cmp.Equal(uint32Data, expected) {
		t.Errorf("Data mismatch:\n%s", cmp.Diff(expected, uint32Data))
	}
}

func TestTwoChannelInt16(t *testing.T) {
	scaler1 := daqmxScalerMetadata(0, 3, 0, 0)
	scaler2 := daqmxScalerMetadata(0, 3, 2, 0)

	metadata := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{4}, [][]byte{scaler1}, 0xFFFFFFFF, nil),
		daqmxChannelMetadata("Channel2", 4, []uint32{4}, [][]byte{scaler2}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 11 00 02 00 12 00 03 00 13 00 04 00 14 00")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel1, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	channel2, err := getChannel(file, "Group", "Channel2")
	if err != nil {
		t.Fatal(err)
	}

	data1, err := channel1.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel1 data: %v", err)
	}

	data2, err := channel2.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel2 data: %v", err)
	}

	int16Data1, ok := data1.([]int16)
	if !ok {
		t.Fatalf("Expected channel1 data type []int16, got %T", data1)
	}

	int16Data2, ok := data2.([]int16)
	if !ok {
		t.Fatalf("Expected channel2 data type []int16, got %T", data2)
	}

	expected1 := []int16{1, 2, 3, 4}
	expected2 := []int16{17, 18, 19, 20}

	if !cmp.Equal(int16Data1, expected1) {
		t.Errorf("Channel1 data mismatch:\n%s", cmp.Diff(expected1, int16Data1))
	}

	if !cmp.Equal(int16Data2, expected2) {
		t.Errorf("Channel2 data mismatch:\n%s", cmp.Diff(expected2, int16Data2))
	}
}

func TestMixedChannelWidths(t *testing.T) {
	scaler1 := daqmxScalerMetadata(0, 1, 0, 0) // Int8, offset 0
	scaler2 := daqmxScalerMetadata(0, 3, 1, 0) // Int16, offset 1
	scaler3 := daqmxScalerMetadata(0, 5, 3, 0) // Int32, offset 3

	metadata := buildMetadata(5,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{7}, [][]byte{scaler1}, 0xFFFFFFFF, nil),
		daqmxChannelMetadata("Channel2", 4, []uint32{7}, [][]byte{scaler2}, 0xFFFFFFFF, nil),
		daqmxChannelMetadata("Channel3", 4, []uint32{7}, [][]byte{scaler3}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes(`
		01 11 00 21 00 00 00
		02 12 00 22 00 00 00
		03 13 00 23 00 00 00
		04 14 00 24 00 00 00
	`)

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel1, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	channel2, err := getChannel(file, "Group", "Channel2")
	if err != nil {
		t.Fatal(err)
	}

	channel3, err := getChannel(file, "Group", "Channel3")
	if err != nil {
		t.Fatal(err)
	}

	data1, err := channel1.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel1 data: %v", err)
	}

	data2, err := channel2.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel2 data: %v", err)
	}

	data3, err := channel3.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel3 data: %v", err)
	}

	int8Data, ok := data1.([]int8)
	if !ok {
		t.Fatalf("Expected channel1 data type []int8, got %T", data1)
	}

	int16Data, ok := data2.([]int16)
	if !ok {
		t.Fatalf("Expected channel2 data type []int16, got %T", data2)
	}

	int32Data, ok := data3.([]int32)
	if !ok {
		t.Fatalf("Expected channel3 data type []int32, got %T", data3)
	}

	expected1 := []int8{1, 2, 3, 4}
	expected2 := []int16{17, 18, 19, 20}
	expected3 := []int32{33, 34, 35, 36}

	if !cmp.Equal(int8Data, expected1) {
		t.Errorf("Channel1 data mismatch:\n%s", cmp.Diff(expected1, int8Data))
	}

	if !cmp.Equal(int16Data, expected2) {
		t.Errorf("Channel2 data mismatch:\n%s", cmp.Diff(expected2, int16Data))
	}

	if !cmp.Equal(int32Data, expected3) {
		t.Errorf("Channel3 data mismatch:\n%s", cmp.Diff(expected3, int32Data))
	}
}

func TestMultipleScalersWithSameType(t *testing.T) {
	scaler1 := daqmxScalerMetadata(0, 3, 0, 0)
	scaler2 := daqmxScalerMetadata(1, 3, 2, 0)

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{4}, [][]byte{scaler1, scaler2}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 11 00 02 00 12 00 03 00 13 00 04 00 14 00")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	// Read data for scaler 0
	data0, err := channel.ReadAll(WithScaling(false), ForDAQmxScaler(0))
	if err != nil {
		t.Fatalf("Failed to read scaler 0 data: %v", err)
	}

	// Read data for scaler 1
	data1, err := channel.ReadAll(WithScaling(false), ForDAQmxScaler(1))
	if err != nil {
		t.Fatalf("Failed to read scaler 1 data: %v", err)
	}

	int16Data0, ok := data0.([]int16)
	if !ok {
		t.Fatalf("Expected scaler 0 data type []int16, got %T", data0)
	}

	int16Data1, ok := data1.([]int16)
	if !ok {
		t.Fatalf("Expected scaler 1 data type []int16, got %T", data1)
	}

	expected0 := []int16{1, 2, 3, 4}
	expected1 := []int16{17, 18, 19, 20}

	if !cmp.Equal(int16Data0, expected0) {
		t.Errorf("Scaler 0 data mismatch:\n%s", cmp.Diff(expected0, int16Data0))
	}

	if !cmp.Equal(int16Data1, expected1) {
		t.Errorf("Scaler 1 data mismatch:\n%s", cmp.Diff(expected1, int16Data1))
	}
}

func TestMultipleRawDataBuffers(t *testing.T) {
	scaler1 := daqmxScalerMetadata(0, 3, 0, 0) // Buffer 0, offset 0
	scaler2 := daqmxScalerMetadata(0, 3, 2, 0) // Buffer 0, offset 2
	scaler3 := daqmxScalerMetadata(0, 3, 0, 1) // Buffer 1, offset 0
	scaler4 := daqmxScalerMetadata(0, 3, 2, 1) // Buffer 1, offset 2

	metadata := buildMetadata(6,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{4, 4}, [][]byte{scaler1}, 0xFFFFFFFF, nil),
		daqmxChannelMetadata("Channel2", 4, []uint32{4, 4}, [][]byte{scaler2}, 0xFFFFFFFF, nil),
		daqmxChannelMetadata("Channel3", 4, []uint32{4, 4}, [][]byte{scaler3}, 0xFFFFFFFF, nil),
		daqmxChannelMetadata("Channel4", 4, []uint32{4, 4}, [][]byte{scaler4}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes(`
		01 00 02 00 03 00 04 00
		05 00 06 00 07 00 08 00
		09 00 0A 00 0B 00 0C 00
		0D 00 0E 00 0F 00 10 00
	`)

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channels := make([]*Channel, 4)
	for i := 1; i <= 4; i++ {
		channelName := fmt.Sprintf("Channel%d", i)
		ch, err := getChannel(file, "Group", channelName)
		if err != nil {
			t.Fatalf("Failed to get %s: %v", channelName, err)
		}
		channels[i-1] = ch
	}

	expectedData := [][]int16{
		{1, 3, 5, 7},
		{2, 4, 6, 8},
		{9, 11, 13, 15},
		{10, 12, 14, 16},
	}

	for i, ch := range channels {
		data, err := ch.ReadAll(WithScaling(false))
		if err != nil {
			t.Fatalf("Failed to read Channel%d data: %v", i+1, err)
		}

		int16Data, ok := data.([]int16)
		if !ok {
			t.Fatalf("Expected Channel%d data type []int16, got %T", i+1, data)
		}

		if !cmp.Equal(int16Data, expectedData[i]) {
			t.Errorf("Channel%d data mismatch:\n%s", i+1, cmp.Diff(expectedData[i], int16Data))
		}
	}
}

func TestDigitalLineScalerData(t *testing.T) {
	scalerMeta := digitalScalerMetadata(0, 0, 0, 0) // Digital, bit offset 0

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{4}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("00 00 00 00 01 00 00 00 00 00 00 00 01 00 00 00")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	file, err := readTDMSFromBytes(tf.build())
	if err != nil {
		t.Fatalf("Failed to read TDMS file: %v", err)
	}

	channel, err := getChannel(file, "Group", "Channel1")
	if err != nil {
		t.Fatal(err)
	}

	data, err := channel.ReadAll(WithScaling(false))
	if err != nil {
		t.Fatalf("Failed to read channel data: %v", err)
	}

	uint8Data, ok := data.([]uint8)
	if !ok {
		t.Fatalf("Expected data type []uint8, got %T", data)
	}

	expected := []uint8{0, 1, 0, 1}
	if !cmp.Equal(uint8Data, expected) {
		t.Errorf("Data mismatch:\n%s", cmp.Diff(expected, uint8Data))
	}
}

func TestDAQmxDataTypeEnumValues(t *testing.T) {
	tests := []struct {
		name     string
		typeID   DAQmxDataType
		expected uint32
	}{
		{"Uint8", DAQmxDataTypeUint8, 0},
		{"Int8", DAQmxDataTypeInt8, 1},
		{"Uint16", DAQmxDataTypeUint16, 2},
		{"Int16", DAQmxDataTypeInt16, 3},
		{"Uint32", DAQmxDataTypeUint32, 4},
		{"Int32", DAQmxDataTypeInt32, 5},
		{"Uint64", DAQmxDataTypeUint64, 6},
		{"Int64", DAQmxDataTypeInt64, 7},
		{"Float32", DAQmxDataTypeFloat32, 8},
		{"Float64", DAQmxDataTypeFloat64, 9},
		{"Timestamp", DAQmxDataTypeTimestamp, 0xFFFFFFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.typeID) != tt.expected {
				t.Errorf("Expected %s = %d, got %d", tt.name, tt.expected, tt.typeID)
			}
		})
	}
}

func TestRawIndexHeaderConstants(t *testing.T) {
	if rawIndexHeaderFormatChangingScaler != 0x1269 {
		t.Errorf("Expected rawIndexHeaderFormatChangingScaler = 0x1269, got 0x%X", rawIndexHeaderFormatChangingScaler)
	}

	if rawIndexHeaderDigitalLineScaler != 0x126A {
		t.Errorf("Expected rawIndexHeaderDigitalLineScaler = 0x126A, got 0x%X", rawIndexHeaderDigitalLineScaler)
	}
}

// Helper to read TDMS from byte slice
func readTDMSFromBytes(data []byte) (*File, error) {
	reader := bytes.NewReader(data)
	return New(reader, false, int64(len(data)))
}

// Helper to get a channel from file
func getChannel(file *File, groupName, channelName string) (*Channel, error) {
	group, ok := file.Groups[groupName]
	if !ok {
		return nil, fmt.Errorf("group '%s' not found", groupName)
	}

	channel, ok := group.Channels[channelName]
	if !ok {
		return nil, fmt.Errorf("channel '%s' not found in group '%s'", channelName, groupName)
	}

	return &channel, nil
}

// Helper to compare slices with type checking
func compareSlice(t *testing.T, name string, got, want any) {
	t.Helper()
	if !cmp.Equal(got, want) {
		t.Errorf("%s mismatch:\n%s", name, cmp.Diff(want, got))
	}
}
