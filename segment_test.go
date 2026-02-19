package tdms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// newFileWithReader creates a File backed by the given byte slice, positioned at offset 0.
func newFileWithReader(data []byte) *File {
	return &File{
		r:       bytes.NewReader(data),
		size:    int64(len(data)),
		objects: make(map[string]object),
	}
}

func TestReadSegmentLeadIn_ValidNonDAQmx(t *testing.T) {
	toc := tocContainsMetadata | tocContainsRawData | tocContainsNewObjectList
	data := buildLeadIn(toc, 4713, 1000, 200)

	f := newFileWithReader(data)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if !li.containsMetadata {
		t.Error("expected containsMetadata to be true")
	}
	if !li.containsRawData {
		t.Error("expected containsRawData to be true")
	}
	if li.containsDAQMXRawData {
		t.Error("expected containsDAQMXRawData to be false")
	}
	if li.isInterleaved {
		t.Error("expected isInterleaved to be false")
	}
	if li.byteOrder != binary.LittleEndian {
		t.Error("expected little endian byte order")
	}
	if !li.newObjectList {
		t.Error("expected newObjectList to be true")
	}
	if li.nextSegmentOffset != 1000 {
		t.Errorf("expected nextSegmentOffset=1000, got %d", li.nextSegmentOffset)
	}
	if li.rawDataOffset != 200 {
		t.Errorf("expected rawDataOffset=200, got %d", li.rawDataOffset)
	}
}

func TestReadSegmentLeadIn_DAQmxFlag(t *testing.T) {
	toc := tocContainsMetadata | tocContainsRawData | tocContainsNewObjectList | tocContainsDAQMXRawData
	data := buildLeadIn(toc, 4713, 500, 100)

	f := newFileWithReader(data)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if !li.containsDAQMXRawData {
		t.Error("expected containsDAQMXRawData to be true")
	}
	if !li.containsRawData {
		t.Error("expected containsRawData to be true")
	}
	if !li.containsMetadata {
		t.Error("expected containsMetadata to be true")
	}
}

func TestReadSegmentLeadIn_NoRawData(t *testing.T) {
	toc := tocContainsMetadata | tocContainsNewObjectList
	data := buildLeadIn(toc, 4712, 300, 300)

	f := newFileWithReader(data)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if !li.containsMetadata {
		t.Error("expected containsMetadata to be true")
	}
	if li.containsRawData {
		t.Error("expected containsRawData to be false for metadata-only segment")
	}
}

func TestReadSegmentLeadIn_BigEndian(t *testing.T) {
	toc := tocContainsMetadata | tocContainsRawData | tocContainsNewObjectList | tocIsBigEndian
	data := buildLeadIn(toc, 4713, 2000, 500)

	f := newFileWithReader(data)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if li.byteOrder != binary.BigEndian {
		t.Error("expected big endian byte order")
	}
	if li.nextSegmentOffset != 2000 {
		t.Errorf("expected nextSegmentOffset=2000, got %d", li.nextSegmentOffset)
	}
	if li.rawDataOffset != 500 {
		t.Errorf("expected rawDataOffset=500, got %d", li.rawDataOffset)
	}
}

func TestReadSegmentLeadIn_Interleaved(t *testing.T) {
	toc := tocContainsMetadata | tocContainsRawData | tocContainsNewObjectList | tocDataIsInterleaved
	data := buildLeadIn(toc, 4712, 100, 50)

	f := newFileWithReader(data)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if !li.isInterleaved {
		t.Error("expected isInterleaved to be true")
	}
}

func TestReadSegmentLeadIn_AllFlagsCombined(t *testing.T) {
	toc := tocContainsMetadata | tocContainsRawData | tocContainsNewObjectList |
		tocDataIsInterleaved | tocIsBigEndian | tocContainsDAQMXRawData
	data := buildLeadIn(toc, 4713, 9999, 1234)

	f := newFileWithReader(data)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if !li.containsMetadata {
		t.Error("containsMetadata should be true")
	}
	if !li.containsRawData {
		t.Error("containsRawData should be true")
	}
	if !li.containsDAQMXRawData {
		t.Error("containsDAQMXRawData should be true")
	}
	if !li.isInterleaved {
		t.Error("isInterleaved should be true")
	}
	if li.byteOrder != binary.BigEndian {
		t.Error("expected big endian")
	}
	if !li.newObjectList {
		t.Error("newObjectList should be true")
	}
}

func TestReadSegmentLeadIn_InvalidMagicBytes(t *testing.T) {
	data := make([]byte, 28)
	copy(data[:4], []byte("XXXX"))

	f := newFileWithReader(data)
	_, err := f.readSegmentLeadIn()

	if err == nil {
		t.Fatal("expected error for invalid magic bytes")
	}
	if !errors.Is(err, ErrInvalidFileFormat) {
		t.Errorf("expected ErrInvalidFileFormat, got: %v", err)
	}
}

func TestReadSegmentLeadIn_IndexFileMagicBytes(t *testing.T) {
	data := make([]byte, 28)
	copy(data[:4], tdmsIndexMagicBytes)
	// Write a valid TOC and version
	binary.LittleEndian.PutUint32(data[4:], tocContainsMetadata|tocContainsNewObjectList)
	binary.LittleEndian.PutUint32(data[8:], 4712)
	binary.LittleEndian.PutUint64(data[12:], 100)
	binary.LittleEndian.PutUint64(data[20:], 50)

	f := newFileWithReader(data)
	f.isIndex = true
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if !li.containsMetadata {
		t.Error("expected containsMetadata to be true for index file")
	}
}

func TestReadSegmentLeadIn_IndexFileMagicBytesRejectsDataMagic(t *testing.T) {
	data := make([]byte, 28)
	copy(data[:4], tdmsMagicBytes)
	binary.LittleEndian.PutUint32(data[4:], tocContainsMetadata)
	binary.LittleEndian.PutUint32(data[8:], 4712)

	f := newFileWithReader(data)
	f.isIndex = true
	_, err := f.readSegmentLeadIn()

	if err == nil {
		t.Fatal("expected error when data magic bytes used for index file")
	}
	if !errors.Is(err, ErrInvalidFileFormat) {
		t.Errorf("expected ErrInvalidFileFormat, got: %v", err)
	}
}

func TestReadSegmentLeadIn_UnsupportedVersion(t *testing.T) {
	toc := tocContainsMetadata | tocContainsNewObjectList
	// Version 1234 is unsupported; only 4712 and 4713 are valid.
	data := buildLeadIn(toc, 1234, 100, 50)

	f := newFileWithReader(data)
	_, err := f.readSegmentLeadIn()

	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion, got: %v", err)
	}
}

func TestReadSegmentLeadIn_Version4712(t *testing.T) {
	toc := tocContainsMetadata | tocContainsNewObjectList
	data := buildLeadIn(toc, 4712, 100, 50)

	f := newFileWithReader(data)
	_, err := f.readSegmentLeadIn()
	requireNoError(t, err)
}

func TestReadSegmentLeadIn_Version4713(t *testing.T) {
	toc := tocContainsMetadata | tocContainsNewObjectList
	data := buildLeadIn(toc, 4713, 100, 50)

	f := newFileWithReader(data)
	_, err := f.readSegmentLeadIn()
	requireNoError(t, err)
}

func TestReadSegmentLeadIn_TruncatedInput(t *testing.T) {
	data := []byte("TDSm") // Only 4 bytes, need 28

	f := newFileWithReader(data)
	_, err := f.readSegmentLeadIn()

	if err == nil {
		t.Fatal("expected error for truncated input")
	}
}

func TestReadSegmentMetadata_SingleObjectWithRawData(t *testing.T) {
	// Build a complete file segment with a single int32 channel containing 3 values
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Voltage", DataTypeInt32, 3, nil),
	)

	rawData := int32Bytes(100, 200, 300)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)

	// Read lead-in first
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	// Should have 3 objects: root, group, channel
	if len(m.objects) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(m.objects))
	}

	channelPath := "/'" + defaultGroupName + "'/'Voltage'"
	obj, ok := m.objects[channelPath]
	if !ok {
		t.Fatalf("channel object %q not found in metadata", channelPath)
	}

	if obj.index == nil {
		t.Fatal("expected non-nil object index for channel")
	}
	if obj.index.dataType != DataTypeInt32 {
		t.Errorf("expected data type Int32, got %s", obj.index.dataType)
	}
	if obj.index.numValues != 3 {
		t.Errorf("expected numValues=3, got %d", obj.index.numValues)
	}
	// totalSize should be 3 * 4 = 12
	if obj.index.totalSize != 12 {
		t.Errorf("expected totalSize=12, got %d", obj.index.totalSize)
	}
}

func TestReadSegmentMetadata_NoRawDataIndex(t *testing.T) {
	// Root and group objects have rawDataIndex=0xFFFFFFFF (no raw data).
	// We include a channel with actual data so that chunkSize > 0 and the
	// metadata parser doesn't hit a divide-by-zero when computing numChunks.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Dummy", DataTypeInt32, 1, nil),
	)

	rawData := int32Bytes(42)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	// Root object should exist with no index
	rootObj, ok := m.objects["/"]
	if !ok {
		t.Fatal("root object not found")
	}
	if rootObj.index != nil {
		t.Error("root object should have nil index")
	}

	// Group object should exist with no index
	groupObj, ok := m.objects["/'Group'"]
	if !ok {
		t.Fatal("group object not found")
	}
	if groupObj.index != nil {
		t.Error("group object should have nil index")
	}
}

func TestReadSegmentMetadata_MatchesPreviousSegment(t *testing.T) {
	// Build a two-segment file where the second segment uses rawDataIndex=0x00000000
	firstMeta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Temp", DataTypeInt32, 3, nil),
	)
	firstRawData := int32Bytes(10, 20, 30)

	secondMeta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadataMatchesPrevious("Temp", nil),
	)
	secondRawData := int32Bytes(40, 50, 60)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), firstMeta, firstRawData)
	// Second segment does NOT have new object list set
	secondTOC := tocContainsMetadata | tocContainsRawData
	tf.addSegment(secondTOC, secondMeta, secondRawData)

	fileBytes := tf.build()
	f := newFileWithReader(fileBytes)

	// Read first segment
	li1, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m1, err := f.readSegmentMetadata(0, li1, nil)
	requireNoError(t, err)

	seg1 := &segment{offset: 0, leadIn: li1, metadata: m1}

	// Seek to second segment
	secondOffset := int64(leadInSize) + int64(li1.nextSegmentOffset)
	_, err = f.r.Seek(secondOffset, 0)
	requireNoError(t, err)

	li2, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m2, err := f.readSegmentMetadata(secondOffset, li2, seg1)
	requireNoError(t, err)

	channelPath := "/'" + defaultGroupName + "'/'Temp'"
	obj, ok := m2.objects[channelPath]
	if !ok {
		t.Fatal("channel object not found in second segment")
	}

	// Index should be inherited from previous segment
	if obj.index == nil {
		t.Fatal("expected index to be inherited from previous segment")
	}
	if obj.index.dataType != DataTypeInt32 {
		t.Errorf("expected inherited data type Int32, got %s", obj.index.dataType)
	}
	if obj.index.numValues != 3 {
		t.Errorf("expected inherited numValues=3, got %d", obj.index.numValues)
	}

}

func TestReadSegmentMetadata_DAQmxFormatChangingScaler(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 3, 0, 0) // scaleID=0, Int16, offset=0, buffer=0

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("DAQChannel", 10, []uint32{2}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := make([]byte, 20) // 10 values * 2 bytes each

	tf := newTestFile()
	tf.addSegment(segmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	channelPath := "/'Group'/'DAQChannel'"
	obj, ok := m.objects[channelPath]
	if !ok {
		t.Fatal("DAQmx channel not found")
	}

	if obj.index == nil {
		t.Fatal("expected non-nil index for DAQmx channel")
	}
	if obj.index.daqmxScalerType != daqmxScalerTypeFormatChanging {
		t.Errorf("expected format changing scaler type, got %d", obj.index.daqmxScalerType)
	}
	if len(obj.index.daqmxScalers) != 1 {
		t.Fatalf("expected 1 DAQmx scaler, got %d", len(obj.index.daqmxScalers))
	}
	scaler, ok := obj.index.daqmxScalers[0]
	if !ok {
		t.Fatal("scaler with scaleID=0 not found")
	}
	if scaler.dataType != DAQmxDataTypeInt16 {
		t.Errorf("expected DAQmx data type Int16 (3), got %d", scaler.dataType)
	}
	if scaler.rawBufferIndex != 0 {
		t.Errorf("expected rawBufferIndex=0, got %d", scaler.rawBufferIndex)
	}
	if scaler.offsetWithinStride != 0 {
		t.Errorf("expected offsetWithinStride=0, got %d", scaler.offsetWithinStride)
	}
	if len(obj.index.daqmxBufferWidths) != 1 {
		t.Fatalf("expected 1 buffer width, got %d", len(obj.index.daqmxBufferWidths))
	}
	if obj.index.daqmxBufferWidths[0] != 2 {
		t.Errorf("expected buffer width 2, got %d", obj.index.daqmxBufferWidths[0])
	}
}

func TestReadSegmentMetadata_DAQmxDigitalLineScaler(t *testing.T) {
	scalerMeta := digitalScalerMetadata(0, 0, 3, 0) // scaleID=0, Uint8, bitOffset=3, buffer=0

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("DigitalCh", 8, []uint32{4}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := make([]byte, 32) // 8 values * 4 bytes per sample width

	tf := newTestFile()
	tf.addSegment(segmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	channelPath := "/'Group'/'DigitalCh'"
	obj, ok := m.objects[channelPath]
	if !ok {
		t.Fatal("digital line channel not found")
	}

	if obj.index.daqmxScalerType != daqmxScalerTypeDigitalLine {
		t.Errorf("expected digital line scaler type, got %d", obj.index.daqmxScalerType)
	}

	scaler, ok := obj.index.daqmxScalers[0]
	if !ok {
		t.Fatal("scaler with scaleID=0 not found")
	}
	if scaler.offsetWithinStride != 3 {
		t.Errorf("expected offsetWithinStride=3 (bit offset), got %d", scaler.offsetWithinStride)
	}
	if scaler.dataType != DAQmxDataTypeUint8 {
		t.Errorf("expected DAQmx data type Uint8 (0), got %d", scaler.dataType)
	}
}

func TestReadSegmentMetadata_MultipleObjectsChunkCalculation(t *testing.T) {
	// Two int32 channels, each with 5 values. Chunk size = 5*4 + 5*4 = 40.
	// Raw data = 80 bytes = 2 chunks.
	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Ch1", DataTypeInt32, 5, nil),
		channelMetadata("Ch2", DataTypeInt32, 5, nil),
	)

	rawData := make([]byte, 80)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	expectedChunkSize := uint64(40) // 20 bytes per channel * 2 channels? No: 5*4=20 per channel, 40 per chunk
	if m.chunkSize != expectedChunkSize {
		t.Errorf("expected chunkSize=%d, got %d", expectedChunkSize, m.chunkSize)
	}

	if m.numChunks != 2 {
		t.Errorf("expected numChunks=2, got %d", m.numChunks)
	}
}

func TestReadSegmentMetadata_ObjectProperties(t *testing.T) {
	// Note: writeProperty maps uint32 → type code 0x03 (DataTypeInt32), so the
	// value is readable via AsInt32(). We use uint32 here intentionally.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Sensor", DataTypeInt32, 2, map[string]any{
			"unit": uint32(42),
		}),
	)

	rawData := int32Bytes(1, 2)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	channelPath := "/'" + defaultGroupName + "'/'Sensor'"
	obj, ok := m.objects[channelPath]
	if !ok {
		t.Fatal("channel with properties not found")
	}

	if len(obj.properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(obj.properties))
	}

	prop, ok := obj.properties["unit"]
	if !ok {
		t.Fatal("property 'unit' not found")
	}
	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 42 {
		t.Errorf("expected property value 42, got %d", val)
	}
}

func TestReadSegmentMetadata_ObjectOrder(t *testing.T) {
	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Alpha", DataTypeInt32, 2, nil),
		channelMetadata("Beta", DataTypeInt32, 2, nil),
	)

	rawData := make([]byte, 16) // 2 channels * 2 values * 4 bytes

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	if len(m.objectOrder) != 4 {
		t.Fatalf("expected 4 objects in order, got %d", len(m.objectOrder))
	}

	expectedOrder := []string{
		"/",
		"/'" + defaultGroupName + "'",
		"/'" + defaultGroupName + "'/'Alpha'",
		"/'" + defaultGroupName + "'/'Beta'",
	}

	if !cmp.Equal(m.objectOrder, expectedOrder) {
		t.Errorf("object order mismatch:\n%s", cmp.Diff(expectedOrder, m.objectOrder))
	}
}

func TestReadSegmentMetadata_DataOffsetCalculation(t *testing.T) {
	// Verify that the offset field on each object index is correctly computed
	// as the absolute position in the file where data for that channel starts.
	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("First", DataTypeInt32, 3, nil),
		channelMetadata("Second", DataTypeInt16, 4, nil),
	)

	firstData := int32Bytes(1, 2, 3)         // 12 bytes
	secondData := int16Bytes(10, 20, 30, 40) // 8 bytes
	var rawData bytes.Buffer
	rawData.Write(firstData)
	rawData.Write(secondData)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData.Bytes())
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	firstPath := "/'" + defaultGroupName + "'/'First'"
	secondPath := "/'" + defaultGroupName + "'/'Second'"

	firstObj := m.objects[firstPath]
	secondObj := m.objects[secondPath]

	if firstObj.index == nil || secondObj.index == nil {
		t.Fatal("expected non-nil indices")
	}

	// Data offset for first channel should be at leadInSize + rawDataOffset
	expectedFirstOffset := int64(leadInSize) + int64(li.rawDataOffset)
	if firstObj.index.offset != expectedFirstOffset {
		t.Errorf("first channel offset: expected %d, got %d", expectedFirstOffset, firstObj.index.offset)
	}

	// Data offset for second channel should be first's offset + first's totalSize
	expectedSecondOffset := expectedFirstOffset + int64(firstObj.index.totalSize)
	if secondObj.index.offset != expectedSecondOffset {
		t.Errorf("second channel offset: expected %d, got %d", expectedSecondOffset, secondObj.index.offset)
	}
}

func TestReadSegmentMetadata_StrideCalculation(t *testing.T) {
	// With two channels of sizes 12 and 8, chunkSize = 20.
	// Stride for first channel = chunkSize - totalSize = 20 - 12 = 8
	// Stride for second channel = 20 - 8 = 12
	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("A", DataTypeInt32, 3, nil), // 12 bytes
		channelMetadata("B", DataTypeInt16, 4, nil), // 8 bytes
	)

	rawData := make([]byte, 20)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	aPath := "/'" + defaultGroupName + "'/'A'"
	bPath := "/'" + defaultGroupName + "'/'B'"

	aObj := m.objects[aPath]
	bObj := m.objects[bPath]

	if aObj.index.stride != 8 {
		t.Errorf("expected stride=8 for channel A, got %d", aObj.index.stride)
	}
	if bObj.index.stride != 12 {
		t.Errorf("expected stride=12 for channel B, got %d", bObj.index.stride)
	}
}

func TestReadSegmentMetadata_NewObjectListFalseWithoutPrevSegment(t *testing.T) {
	// When newObjectList is false but there's no previous segment, it should error.
	meta := buildMetadata(1,
		rootMetadata(),
	)

	// TOC without tocContainsNewObjectList
	toc := tocContainsMetadata | tocContainsRawData

	var buf bytes.Buffer
	buf.Write(tdmsMagicBytes)
	_ = binary.Write(&buf, binary.LittleEndian, toc)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4713))
	_ = binary.Write(&buf, binary.LittleEndian, uint64(len(meta)))
	_ = binary.Write(&buf, binary.LittleEndian, uint64(len(meta)))
	buf.Write(meta)

	f := newFileWithReader(buf.Bytes())
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	_, err = f.readSegmentMetadata(0, li, nil)
	if err == nil {
		t.Fatal("expected error when newObjectList=false and prevSegment=nil")
	}
}

func TestReadSegmentMetadata_MatchesPreviousWithoutPrevSegment(t *testing.T) {
	// Object with rawDataIndex=0x00000000 but no previous segment should fail
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadataMatchesPrevious("Ch1", nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, make([]byte, 12))
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	_, err = f.readSegmentMetadata(0, li, nil)
	if err == nil {
		t.Fatal("expected error when rawDataIndex matches previous but no previous segment exists")
	}
}

func TestReadSegmentMetadata_MultipleDAQmxScalers(t *testing.T) {
	scaler1 := daqmxScalerMetadata(0, 3, 0, 0) // Int16, offset=0
	scaler2 := daqmxScalerMetadata(1, 3, 2, 0) // Int16, offset=2

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("MultiScaler", 5, []uint32{4}, [][]byte{scaler1, scaler2}, 0xFFFFFFFF, nil),
	)

	rawData := make([]byte, 20) // 5 values * 4 bytes buffer width

	tf := newTestFile()
	tf.addSegment(segmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	channelPath := "/'Group'/'MultiScaler'"
	obj := m.objects[channelPath]

	if len(obj.index.daqmxScalers) != 2 {
		t.Fatalf("expected 2 scalers, got %d", len(obj.index.daqmxScalers))
	}

	s0, ok := obj.index.daqmxScalers[0]
	if !ok {
		t.Fatal("scaler with scaleID=0 not found")
	}
	if s0.offsetWithinStride != 0 {
		t.Errorf("scaler 0 offset: expected 0, got %d", s0.offsetWithinStride)
	}

	s1, ok := obj.index.daqmxScalers[1]
	if !ok {
		t.Fatal("scaler with scaleID=1 not found")
	}
	if s1.offsetWithinStride != 2 {
		t.Errorf("scaler 1 offset: expected 2, got %d", s1.offsetWithinStride)
	}
}

func TestReadSegmentMetadata_MultipleBufferWidths(t *testing.T) {
	scaler := daqmxScalerMetadata(0, 3, 0, 0)

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("DualBuf", 4, []uint32{4, 8}, [][]byte{scaler}, 0xFFFFFFFF, nil),
	)

	rawData := make([]byte, 48) // buffers: 4*4=16 + 8*4=32 = 48

	tf := newTestFile()
	tf.addSegment(segmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	channelPath := "/'Group'/'DualBuf'"
	obj := m.objects[channelPath]

	if len(obj.index.daqmxBufferWidths) != 2 {
		t.Fatalf("expected 2 buffer widths, got %d", len(obj.index.daqmxBufferWidths))
	}
	if obj.index.daqmxBufferWidths[0] != 4 {
		t.Errorf("buffer width[0]: expected 4, got %d", obj.index.daqmxBufferWidths[0])
	}
	if obj.index.daqmxBufferWidths[1] != 8 {
		t.Errorf("buffer width[1]: expected 8, got %d", obj.index.daqmxBufferWidths[1])
	}
}

func TestReadSegmentMetadata_DimensionMustBeOne(t *testing.T) {
	// Build channel metadata with dimension != 1 (which is invalid in TDMS v2)
	path := "/'" + defaultGroupName + "'/'BadDim'"
	var metaObj bytes.Buffer
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(len(path)))
	metaObj.WriteString(path)
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(0x14)) // normal raw data index
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(DataTypeInt32))
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(2)) // dimension = 2 (invalid!)
	_ = binary.Write(&metaObj, binary.LittleEndian, uint64(5))
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(0)) // no props

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		metaObj.Bytes(),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, make([]byte, 20))
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	_, err = f.readSegmentMetadata(0, li, nil)
	if err == nil {
		t.Fatal("expected error for dimension != 1")
	}
	if !errors.Is(err, ErrInvalidFileFormat) {
		t.Errorf("expected ErrInvalidFileFormat, got: %v", err)
	}
}

func TestReadSegmentMetadata_InterleavedStringIsRejected(t *testing.T) {
	// Interleaved segments with string (variable-width) data types should fail.
	path := "/'" + defaultGroupName + "'/'Str'"
	var metaObj bytes.Buffer
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(len(path)))
	metaObj.WriteString(path)
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(0x14))
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(DataTypeString))
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(1))
	_ = binary.Write(&metaObj, binary.LittleEndian, uint64(2))
	_ = binary.Write(&metaObj, binary.LittleEndian, uint64(10)) // totalSize
	_ = binary.Write(&metaObj, binary.LittleEndian, uint32(0))  // no props

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		metaObj.Bytes(),
	)

	// Use interleaved TOC
	toc := tocContainsMetadata | tocContainsRawData | tocContainsNewObjectList | tocDataIsInterleaved
	var buf bytes.Buffer
	buf.Write(tdmsMagicBytes)
	_ = binary.Write(&buf, binary.LittleEndian, toc)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4713))
	nextOffset := uint64(len(meta) + 10)
	rawOffset := uint64(len(meta))
	_ = binary.Write(&buf, binary.LittleEndian, nextOffset)
	_ = binary.Write(&buf, binary.LittleEndian, rawOffset)
	buf.Write(meta)
	buf.Write(make([]byte, 10))

	f := newFileWithReader(buf.Bytes())
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	_, err = f.readSegmentMetadata(0, li, nil)
	if err == nil {
		t.Fatal("expected error for interleaved string data")
	}
	if !errors.Is(err, ErrInvalidFileFormat) {
		t.Errorf("expected ErrInvalidFileFormat, got: %v", err)
	}
}

func TestReadSegmentMetadata_VariableSizeDataType(t *testing.T) {
	// String data type requires explicit totalSize in the raw data index
	totalStringBytes := uint64(15) // total raw string data size in the chunk

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadataWithTotalSize("Strings", DataTypeString, 3, totalStringBytes, nil),
	)

	// For string channels, totalSize includes offset bytes + string bytes
	rawData := make([]byte, totalStringBytes)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	channelPath := "/'" + defaultGroupName + "'/'Strings'"
	obj := m.objects[channelPath]

	if obj.index.dataType != DataTypeString {
		t.Errorf("expected DataTypeString, got %s", obj.index.dataType)
	}
	if obj.index.totalSize != totalStringBytes {
		t.Errorf("expected totalSize=%d, got %d", totalStringBytes, obj.index.totalSize)
	}
	if obj.index.numValues != 3 {
		t.Errorf("expected numValues=3, got %d", obj.index.numValues)
	}
}

func TestReadSegmentMetadata_SingleChunkCalculation(t *testing.T) {
	// One channel with 4 int16 values. Total raw = 8 bytes. Chunk = 8 bytes. numChunks = 1.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("OnlyChannel", DataTypeInt16, 4, nil),
	)

	rawData := int16Bytes(1, 2, 3, 4)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	if m.chunkSize != 8 {
		t.Errorf("expected chunkSize=8, got %d", m.chunkSize)
	}
	if m.numChunks != 1 {
		t.Errorf("expected numChunks=1, got %d", m.numChunks)
	}
}

func TestReadSegmentMetadata_MultipleChunksCalculation(t *testing.T) {
	// One int32 channel with 2 values per chunk. Raw data = 24 bytes = 3 chunks.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Chunked", DataTypeInt32, 2, nil),
	)

	rawData := make([]byte, 24) // 3 chunks * 2 values * 4 bytes

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	if m.chunkSize != 8 {
		t.Errorf("expected chunkSize=8, got %d", m.chunkSize)
	}
	if m.numChunks != 3 {
		t.Errorf("expected numChunks=3, got %d", m.numChunks)
	}
}

func TestReadSegmentMetadata_PropertiesMergedOnDuplicate(t *testing.T) {
	// When the same object path appears twice in the metadata, properties
	// should be merged and new raw data index should replace the old.
	// We test this by building two separate metadata entries for the same
	// channel in a single metadata block.
	//
	// Note: writeProperty maps uint32 → type code 0x03 (DataTypeInt32), so
	// we use uint32 values here to get correct 4-byte property reads.
	channelPath := "/'" + defaultGroupName + "'/'Merged'"

	obj1 := channelMetadata("Merged", DataTypeInt32, 5, map[string]any{"prop1": uint32(1)})
	obj2 := channelMetadata("Merged", DataTypeInt32, 10, map[string]any{"prop2": uint32(2)})

	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		obj1,
		obj2,
	)

	rawData := make([]byte, 40) // 10 values * 4 bytes (uses the latest numValues)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	f := newFileWithReader(fileBytes)
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	obj, ok := m.objects[channelPath]
	if !ok {
		t.Fatal("merged channel not found")
	}

	// Both properties should be present
	if len(obj.properties) != 2 {
		t.Errorf("expected 2 merged properties, got %d", len(obj.properties))
	}
	if _, ok := obj.properties["prop1"]; !ok {
		t.Error("prop1 missing after merge")
	}
	if _, ok := obj.properties["prop2"]; !ok {
		t.Error("prop2 missing after merge")
	}

	// The index should reflect the second object's values (numValues=10)
	if obj.index.numValues != 10 {
		t.Errorf("expected numValues=10 after merge, got %d", obj.index.numValues)
	}
}

func TestReadSegmentMetadata_IncompleteSegment(t *testing.T) {
	// When nextSegmentOffset is 0xFFFFFFFFFFFFFFFF, the segment is incomplete.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Incomplete", DataTypeInt32, 5, nil),
	)

	rawData := make([]byte, 20) // 5 * 4

	toc := standardSegmentTOC()
	var buf bytes.Buffer
	buf.Write(tdmsMagicBytes)
	_ = binary.Write(&buf, binary.LittleEndian, toc)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4713))
	_ = binary.Write(&buf, binary.LittleEndian, uint64(0xFFFFFFFFFFFFFFFF)) // incomplete
	_ = binary.Write(&buf, binary.LittleEndian, uint64(len(meta)))
	buf.Write(meta)
	buf.Write(rawData)

	f := newFileWithReader(buf.Bytes())
	li, err := f.readSegmentLeadIn()
	requireNoError(t, err)

	if li.nextSegmentOffset != segmentIncomplete {
		t.Errorf("expected segmentIncomplete, got %d", li.nextSegmentOffset)
	}

	m, err := f.readSegmentMetadata(0, li, nil)
	requireNoError(t, err)

	// Even with incomplete segment, metadata should parse and chunks calculated
	// based on remaining file size.
	if m.numChunks != 1 {
		t.Errorf("expected numChunks=1 for incomplete segment, got %d", m.numChunks)
	}
}
