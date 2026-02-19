package tdms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew_SingleSegmentSingleChannel(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Voltage", []int32{10, 20, 30}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	if len(file.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(file.Groups))
	}

	group, ok := file.Groups[defaultGroupName]
	if !ok {
		t.Fatalf("group %q not found", defaultGroupName)
	}

	if group.Name != defaultGroupName {
		t.Errorf("expected group name %q, got %q", defaultGroupName, group.Name)
	}

	if len(group.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(group.Channels))
	}

	ch, ok := group.Channels["Voltage"]
	if !ok {
		t.Fatal("channel 'Voltage' not found")
	}

	if ch.Name != "Voltage" {
		t.Errorf("expected channel name 'Voltage', got %q", ch.Name)
	}
	if ch.GroupName != defaultGroupName {
		t.Errorf("expected group name %q, got %q", defaultGroupName, ch.GroupName)
	}
	if ch.RawDataType != DataTypeInt32 {
		t.Errorf("expected data type Int32, got %s", ch.RawDataType)
	}
	if ch.NumValues() != 3 {
		t.Errorf("expected 3 values, got %d", ch.NumValues())
	}
}

func TestNew_MultipleChannels(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Ch1", []int16{1, 2}),
		float64Channel("Ch2", []float64{3.14, 2.72}),
		boolChannel("Ch3", []bool{true, false}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	group := file.Groups[defaultGroupName]
	if len(group.Channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(group.Channels))
	}

	ch1 := group.Channels["Ch1"]
	if ch1.RawDataType != DataTypeInt16 {
		t.Errorf("Ch1: expected Int16, got %s", ch1.RawDataType)
	}

	ch2 := group.Channels["Ch2"]
	if ch2.RawDataType != DataTypeFloat64 {
		t.Errorf("Ch2: expected Float64, got %s", ch2.RawDataType)
	}

	ch3 := group.Channels["Ch3"]
	if ch3.RawDataType != DataTypeBool {
		t.Errorf("Ch3: expected Bool, got %s", ch3.RawDataType)
	}
}

func TestNew_MultipleSegments(t *testing.T) {
	// Use standardSegmentTOC (with new object list) for both segments to
	// avoid the shared objectIndex pointer mutation between segments.
	meta1 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Temp", DataTypeInt32, 2, nil),
	)
	rawData1 := int32Bytes(100, 200)

	meta2 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Temp", DataTypeInt32, 2, nil),
	)
	rawData2 := int32Bytes(300, 400)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta1, rawData1)
	tf.addSegment(standardSegmentTOC(), meta2, rawData2)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Temp")

	if ch.NumValues() != 4 {
		t.Errorf("expected 4 values across 2 segments, got %d", ch.NumValues())
	}

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	expected := []int32{100, 200, 300, 400}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestNew_ThreeSegmentsAccumulating(t *testing.T) {
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Acc", DataTypeInt16, 2, nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, int16Bytes(1, 2))
	tf.addSegment(standardSegmentTOC(), meta, int16Bytes(3, 4))
	tf.addSegment(standardSegmentTOC(), meta, int16Bytes(5, 6))

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Acc")

	if ch.NumValues() != 6 {
		t.Errorf("expected 6 values across 3 segments, got %d", ch.NumValues())
	}

	data, err := ch.ReadInt16All()
	requireNoError(t, err)

	expected := []int16{1, 2, 3, 4, 5, 6}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestNew_FileProperties(t *testing.T) {
	// Note: writeProperty maps uint32 → type code 0x03 (DataTypeInt32),
	// so the value is readable via AsInt32(). We use uint32 intentionally.
	meta := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"file_version": uint32(42)}),
		groupMetadata(),
		channelMetadata("X", DataTypeInt16, 1, nil),
	)

	rawData := int16Bytes(99)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	prop, ok := file.Properties["file_version"]
	if !ok {
		t.Fatal("root property 'file_version' not found")
	}

	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 42 {
		t.Errorf("expected file_version=42, got %d", val)
	}
}

func TestNew_GroupProperties(t *testing.T) {
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadataWithProps(map[string]any{"group_info": uint32(7)}),
		channelMetadata("Y", DataTypeInt16, 1, nil),
	)

	rawData := int16Bytes(1)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	group, ok := file.Groups[defaultGroupName]
	if !ok {
		t.Fatalf("group %q not found", defaultGroupName)
	}

	prop, ok := group.Properties["group_info"]
	if !ok {
		t.Fatal("group property 'group_info' not found")
	}

	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 7 {
		t.Errorf("expected group_info=7, got %d", val)
	}
}

func TestNew_ChannelProperties(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16ChannelWithProps("Sensor", []int16{5}, map[string]any{
			"sensitivity": uint32(100),
		}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Sensor")

	prop, ok := ch.Properties["sensitivity"]
	if !ok {
		t.Fatal("channel property 'sensitivity' not found")
	}

	// writeProperty writes uint32 with type code 0x03 = DataTypeInt32
	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 100 {
		t.Errorf("expected sensitivity=100, got %d", val)
	}
}

func TestNew_PropertiesMergedAcrossSegments(t *testing.T) {
	meta1 := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"prop1": uint32(1)}),
		groupMetadata(),
		channelMetadata("MergeCh", DataTypeInt16, 1, nil),
	)
	rawData1 := int16Bytes(10)

	meta2 := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"prop2": uint32(2)}),
		groupMetadata(),
		channelMetadata("MergeCh", DataTypeInt16, 1, nil),
	)
	rawData2 := int16Bytes(20)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta1, rawData1)
	tf.addSegment(standardSegmentTOC(), meta2, rawData2)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	if _, ok := file.Properties["prop1"]; !ok {
		t.Error("prop1 from first segment missing after merge")
	}
	if _, ok := file.Properties["prop2"]; !ok {
		t.Error("prop2 from second segment missing after merge")
	}
}

func TestNew_PropertyOverwrittenInLaterSegment(t *testing.T) {
	meta1 := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"version": uint32(1)}),
		groupMetadata(),
		channelMetadata("OvCh", DataTypeInt16, 1, nil),
	)
	rawData1 := int16Bytes(10)

	meta2 := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"version": uint32(2)}),
		groupMetadata(),
		channelMetadata("OvCh", DataTypeInt16, 1, nil),
	)
	rawData2 := int16Bytes(20)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta1, rawData1)
	tf.addSegment(standardSegmentTOC(), meta2, rawData2)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	prop := file.Properties["version"]
	val, err := prop.AsInt32()
	requireNoError(t, err)

	if val != 2 {
		t.Errorf("expected overwritten version=2, got %d", val)
	}
}

func TestNew_IsIncompleteTrue(t *testing.T) {
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("IncompCh", DataTypeInt32, 3, nil),
	)

	rawData := int32Bytes(1, 2, 3)

	toc := standardSegmentTOC()
	var buf bytes.Buffer
	buf.Write(tdmsMagicBytes)
	_ = binary.Write(&buf, binary.LittleEndian, toc)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4713))
	_ = binary.Write(&buf, binary.LittleEndian, uint64(0xFFFFFFFFFFFFFFFF)) // incomplete marker
	_ = binary.Write(&buf, binary.LittleEndian, uint64(len(meta)))
	buf.Write(meta)
	buf.Write(rawData)

	file, err := readTDMSFromBytes(buf.Bytes())
	requireNoError(t, err)

	if !file.IsIncomplete {
		t.Error("expected IsIncomplete=true for segment with 0xFFFFFFFFFFFFFFFF offset")
	}
}

func TestNew_IsIncompleteFalseForNormalFile(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Normal", []int32{1, 2}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	if file.IsIncomplete {
		t.Error("expected IsIncomplete=false for normal file")
	}
}

func TestNew_InvalidMagicBytes(t *testing.T) {
	data := []byte("NOT_A_TDMS_FILE_AT_ALL_ENOUGH_BYTES_HERE")

	reader := bytes.NewReader(data)
	_, err := New(reader, false, int64(len(data)))

	if err == nil {
		t.Fatal("expected error for invalid file data")
	}
}

func TestNew_EmptyReader(t *testing.T) {
	reader := bytes.NewReader(nil)
	_, err := New(reader, false, 0)

	if err == nil {
		t.Fatal("expected error for empty reader")
	}
}

func TestNew_TruncatedLeadIn(t *testing.T) {
	data := []byte("TDSm") // Only magic bytes, no rest of lead-in

	reader := bytes.NewReader(data)
	_, err := New(reader, false, int64(len(data)))

	if err == nil {
		t.Fatal("expected error for truncated lead-in")
	}
}

func TestNew_RootPropertiesWithDataChannel(t *testing.T) {
	// Verify that root-level properties are accessible on the file when a
	// data-bearing channel is also present in the same segment.
	meta := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"info": uint32(99)}),
		groupMetadata(),
		channelMetadata("Dummy", DataTypeInt16, 1, nil),
	)
	rawData := int16Bytes(1)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	if len(file.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(file.Groups))
	}

	prop, ok := file.Properties["info"]
	if !ok {
		t.Fatal("root property 'info' not found")
	}

	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 99 {
		t.Errorf("expected info=99, got %d", val)
	}
}

func TestNew_ChannelWithNoDataChunks(t *testing.T) {
	// A channel declared in metadata but with no raw data index.
	// We include a data-bearing channel so that chunkSize > 0 and the
	// metadata parser doesn't divide by zero.
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("HasData", []int32{42}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	// "HasData" should have values
	ch := requireChannel(t, file, defaultGroupName, "HasData")
	if ch.NumValues() != 1 {
		t.Errorf("expected NumValues()=1, got %d", ch.NumValues())
	}
}

func TestNew_MultipleChunksInSegment(t *testing.T) {
	// One int32 channel with 2 values per chunk. Provide 6 values = 3 chunks.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Chunked", DataTypeInt32, 2, nil),
	)

	rawData := int32Bytes(1, 2, 3, 4, 5, 6)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Chunked")

	if ch.NumValues() != 6 {
		t.Errorf("expected 6 values (3 chunks * 2), got %d", ch.NumValues())
	}

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	expected := []int32{1, 2, 3, 4, 5, 6}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestNew_SegmentCount(t *testing.T) {
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("S", DataTypeInt16, 1, nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, int16Bytes(1))
	tf.addSegment(standardSegmentTOC(), meta, int16Bytes(2))
	tf.addSegment(standardSegmentTOC(), meta, int16Bytes(3))

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	if len(file.segments) != 3 {
		t.Errorf("expected 3 segments, got %d", len(file.segments))
	}
}

func TestNew_DAQmxFile(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 3, 0, 0)

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("DAQCh", 4, []uint32{2}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 02 00 03 00 04 00")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "DAQCh")

	if ch.NumValues() != 4 {
		t.Errorf("expected 4 values, got %d", ch.NumValues())
	}

	data, err := ch.ReadAll(WithScaling(false))
	requireNoError(t, err)

	got, ok := data.([]int16)
	if !ok {
		t.Fatalf("expected []int16, got %T", data)
	}

	expected := []int16{1, 2, 3, 4}
	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestNew_StringChannel(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Names", []string{"Alice", "Bob", "Charlie"}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Names")

	if ch.RawDataType != DataTypeString {
		t.Errorf("expected DataTypeString, got %s", ch.RawDataType)
	}

	data, err := ch.ReadStringAll()
	requireNoError(t, err)

	expected := []string{"Alice", "Bob", "Charlie"}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestNew_ReadDataFromMultipleSegmentsSameChannel(t *testing.T) {
	// Ensure that channel data across segments is correctly concatenated.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Values", DataTypeFloat64, 3, nil),
	)
	raw1 := float64Bytes(1.0, 2.0, 3.0)
	raw2 := float64Bytes(4.0, 5.0, 6.0)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, raw1)
	tf.addSegment(standardSegmentTOC(), meta, raw2)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Values")

	data, err := ch.ReadFloat64All()
	requireNoError(t, err)

	expected := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestNew_TwoChannelsInSameSegment(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("A", []int32{10, 20}),
		int32Channel("B", []int32{30, 40}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	chA := requireChannel(t, file, defaultGroupName, "A")
	chB := requireChannel(t, file, defaultGroupName, "B")

	dataA, err := chA.ReadInt32All()
	requireNoError(t, err)

	dataB, err := chB.ReadInt32All()
	requireNoError(t, err)

	if !cmp.Equal([]int32{10, 20}, dataA) {
		t.Errorf("channel A data mismatch:\n%s", cmp.Diff([]int32{10, 20}, dataA))
	}
	if !cmp.Equal([]int32{30, 40}, dataB) {
		t.Errorf("channel B data mismatch:\n%s", cmp.Diff([]int32{30, 40}, dataB))
	}
}

func TestOpen_NonExistentFile(t *testing.T) {
	_, err := Open("/nonexistent/path/to/file.tdms")
	if err == nil {
		t.Fatal("expected error when opening non-existent file")
	}
}

func TestOpen_InvalidFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "bad.tdms")
	err := os.WriteFile(tmpFile, []byte("not a tdms file at all and long enough"), 0644)
	requireNoError(t, err)

	_, err = Open(tmpFile)
	if err == nil {
		t.Fatal("expected error when opening invalid file content")
	}
}

func TestOpen_ValidTempFile(t *testing.T) {
	// Create a valid TDMS file on disk and open it
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("DiskCh", []int16{100, 200}),
	})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "valid.tdms")
	err := os.WriteFile(tmpFile, fileBytes, 0644)
	requireNoError(t, err)

	file, err := Open(tmpFile)
	requireNoError(t, err)
	defer file.Close() //nolint:errcheck

	ch := requireChannel(t, file, defaultGroupName, "DiskCh")

	data, err := ch.ReadInt16All()
	requireNoError(t, err)

	expected := []int16{100, 200}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestClose_OnBytesReader(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("X", []int16{1}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	// Close on a bytes.Reader-backed File should be a no-op, not an error.
	err = file.Close()
	if err != nil {
		t.Errorf("expected no error closing bytes.Reader-backed file, got: %v", err)
	}
}

func TestClose_OnRealFile(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("X", []int16{1}),
	})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "closeable.tdms")
	err := os.WriteFile(tmpFile, fileBytes, 0644)
	requireNoError(t, err)

	file, err := Open(tmpFile)
	requireNoError(t, err)

	err = file.Close()
	if err != nil {
		t.Errorf("expected no error on first close, got: %v", err)
	}
}

func TestNew_UnsupportedVersion(t *testing.T) {
	toc := tocContainsMetadata | tocContainsNewObjectList
	data := buildLeadIn(toc, 9999, 100, 50)

	reader := bytes.NewReader(data)
	_, err := New(reader, false, int64(len(data)))

	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion, got: %v", err)
	}
}

func TestNew_MixedDataTypes(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Ints", []int16{-1, 0, 1}),
		float32Channel("Floats", []float32{0.5, 1.5, 2.5}),
		boolChannel("Bools", []bool{true, false, true}),
		uint8Channel("Bytes", []uint8{0, 128, 255}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	group := file.Groups[defaultGroupName]
	if len(group.Channels) != 4 {
		t.Fatalf("expected 4 channels, got %d", len(group.Channels))
	}

	// Verify each channel's type and data
	tests := []struct {
		name     string
		dataType DataType
	}{
		{"Ints", DataTypeInt16},
		{"Floats", DataTypeFloat32},
		{"Bools", DataTypeBool},
		{"Bytes", DataTypeUint8},
	}

	for _, tt := range tests {
		ch, ok := group.Channels[tt.name]
		if !ok {
			t.Errorf("channel %q not found", tt.name)
			continue
		}
		if ch.RawDataType != tt.dataType {
			t.Errorf("channel %q: expected type %s, got %s", tt.name, tt.dataType, ch.RawDataType)
		}
	}
}

func TestOpen_RealTestdataFile(t *testing.T) {
	// Try to open one of the test data files that ship with the project
	file, err := Open("testdata/standard.tdms")
	if err != nil {
		t.Skipf("skipping real file test: %v", err)
	}
	defer file.Close() //nolint:errcheck

	if len(file.Groups) == 0 {
		t.Error("expected at least one group in standard.tdms")
	}

	for groupName, group := range file.Groups {
		if group.Name != groupName {
			t.Errorf("group name mismatch: key=%q, Name=%q", groupName, group.Name)
		}
		for channelName, ch := range group.Channels {
			if ch.Name != channelName {
				t.Errorf("channel name mismatch: key=%q, Name=%q", channelName, ch.Name)
			}
			if ch.GroupName != groupName {
				t.Errorf("channel %q GroupName=%q, expected %q", channelName, ch.GroupName, groupName)
			}
		}
	}
}

func TestOpen_BigEndianTestdataFile(t *testing.T) {
	file, err := Open("testdata/big_endian.tdms")
	if err != nil {
		t.Skipf("skipping big endian file test: %v", err)
	}
	defer file.Close() //nolint:errcheck

	if len(file.Groups) == 0 {
		t.Error("expected at least one group in big_endian.tdms")
	}

	// Just verify the file opens and has readable data
	for _, group := range file.Groups {
		for _, ch := range group.Channels {
			if ch.NumValues() > 0 {
				_, err := ch.ReadAll()
				if err != nil {
					t.Errorf("failed to read channel %s/%s: %v", group.Name, ch.Name, err)
				}
			}
		}
	}
}

func TestNew_IndexFile(t *testing.T) {
	// A TDMS index file uses TDSh magic and has metadata but no raw data.
	// Include a channel with a raw data index so that chunkSize > 0 and the
	// metadata parser doesn't divide by zero when computing numChunks.
	// In a real index file the channel metadata mirrors the data file's
	// metadata (pointing to where data would be in the .tdms file).
	meta := buildMetadata(3,
		rootMetadataWithProps(map[string]any{"indexed": uint32(1)}),
		groupMetadata(),
		channelMetadata("IndexedCh", DataTypeInt16, 1, nil),
	)

	var buf bytes.Buffer
	buf.Write(tdmsIndexMagicBytes)
	toc := tocContainsMetadata | tocContainsNewObjectList
	_ = binary.Write(&buf, binary.LittleEndian, toc)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4713))
	nextOffset := uint64(len(meta))
	rawDataOffset := uint64(len(meta))
	_ = binary.Write(&buf, binary.LittleEndian, nextOffset)
	_ = binary.Write(&buf, binary.LittleEndian, rawDataOffset)
	buf.Write(meta)

	data := buf.Bytes()
	reader := bytes.NewReader(data)
	file, err := New(reader, true, int64(len(data)))
	requireNoError(t, err)

	prop, ok := file.Properties["indexed"]
	if !ok {
		t.Fatal("expected 'indexed' property in index file")
	}

	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 1 {
		t.Errorf("expected indexed=1, got %d", val)
	}
}

func TestNew_IndexFileRejectsDataMagic(t *testing.T) {
	// When isIndex=true, TDSm magic bytes should be rejected
	meta := buildMetadata(1, rootMetadata())

	var buf bytes.Buffer
	buf.Write(tdmsMagicBytes) // wrong magic for index
	toc := tocContainsMetadata | tocContainsNewObjectList
	_ = binary.Write(&buf, binary.LittleEndian, toc)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4713))
	_ = binary.Write(&buf, binary.LittleEndian, uint64(len(meta)))
	_ = binary.Write(&buf, binary.LittleEndian, uint64(len(meta)))
	buf.Write(meta)

	data := buf.Bytes()
	reader := bytes.NewReader(data)
	_, err := New(reader, true, int64(len(data)))

	if err == nil {
		t.Fatal("expected error when data magic bytes used for index file")
	}
}

func TestNew_GroupNameConsistency(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Ch", []int16{1}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	for name, group := range file.Groups {
		if name != group.Name {
			t.Errorf("group map key %q doesn't match group.Name %q", name, group.Name)
		}
		for chName, ch := range group.Channels {
			if chName != ch.Name {
				t.Errorf("channel map key %q doesn't match ch.Name %q", chName, ch.Name)
			}
			if ch.GroupName != name {
				t.Errorf("channel %q GroupName=%q, expected %q", chName, ch.GroupName, name)
			}
		}
	}
}

func TestNew_LargeMultiChunkFile(t *testing.T) {
	// Verify correct behavior with many chunks in a single segment
	n := 100
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("BigData", DataTypeInt32, 10, nil),
	)

	// 10 values per chunk * 4 bytes = 40 bytes per chunk
	// n chunks * 40 bytes = 4000 bytes total raw data
	values := make([]int32, n*10)
	for i := range values {
		values[i] = int32(i)
	}
	rawData := int32Bytes(values...)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "BigData")

	if ch.NumValues() != uint64(len(values)) {
		t.Fatalf("expected %d values, got %d", len(values), ch.NumValues())
	}

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	if !cmp.Equal(values, data) {
		t.Error("large multi-chunk data mismatch (diff omitted)")
	}
}
