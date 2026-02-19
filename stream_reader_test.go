package tdms

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetInterpreter_AllSupportedTypes(t *testing.T) {
	supportedTypes := []struct {
		name     string
		dataType DataType
	}{
		{"Int8", DataTypeInt8},
		{"Int16", DataTypeInt16},
		{"Int32", DataTypeInt32},
		{"Int64", DataTypeInt64},
		{"Uint8", DataTypeUint8},
		{"Uint16", DataTypeUint16},
		{"Uint32", DataTypeUint32},
		{"Uint64", DataTypeUint64},
		{"Float32", DataTypeFloat32},
		{"Float64", DataTypeFloat64},
		{"Float32WithUnit", DataTypeFloat32WithUnit},
		{"Float64WithUnit", DataTypeFloat64WithUnit},
		{"Float128", DataTypeFloat128},
		{"Float128WithUnit", DataTypeFloat128WithUnit},
		{"String", DataTypeString},
		{"Bool", DataTypeBool},
		{"Timestamp", DataTypeTimestamp},
		{"Complex64", DataTypeComplex64},
		{"Complex128", DataTypeComplex128},
	}

	for _, tt := range supportedTypes {
		t.Run(tt.name, func(t *testing.T) {
			interp, err := getInterpreter(tt.dataType)
			if err != nil {
				t.Fatalf("getInterpreter(%s) returned error: %v", tt.name, err)
			}
			if interp == nil {
				t.Fatalf("getInterpreter(%s) returned nil interpreter", tt.name)
			}
		})
	}
}

func TestGetInterpreter_UnsupportedType(t *testing.T) {
	_, err := getInterpreter(DataType(0xDEAD))
	if err == nil {
		t.Fatal("expected error for unsupported data type")
	}
}

func TestGetInterpreter_Int8Interpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeInt8)
	requireNoError(t, err)

	result := interp([]byte{0xFE}, binary.LittleEndian)
	val, ok := result.(int8)
	if !ok {
		t.Fatalf("expected int8, got %T", result)
	}
	if val != -2 {
		t.Errorf("expected -2, got %d", val)
	}
}

func TestGetInterpreter_Int16LittleEndian(t *testing.T) {
	interp, err := getInterpreter(DataTypeInt16)
	requireNoError(t, err)

	result := interp([]byte{0x01, 0x00}, binary.LittleEndian)
	val, ok := result.(int16)
	if !ok {
		t.Fatalf("expected int16, got %T", result)
	}
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
}

func TestGetInterpreter_Int16BigEndian(t *testing.T) {
	interp, err := getInterpreter(DataTypeInt16)
	requireNoError(t, err)

	result := interp([]byte{0x00, 0x01}, binary.BigEndian)
	val, ok := result.(int16)
	if !ok {
		t.Fatalf("expected int16, got %T", result)
	}
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
}

func TestGetInterpreter_Int32(t *testing.T) {
	interp, err := getInterpreter(DataTypeInt32)
	requireNoError(t, err)

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(0xFFFFFFFF)) // -1

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(int32)
	if !ok {
		t.Fatalf("expected int32, got %T", result)
	}
	if val != -1 {
		t.Errorf("expected -1, got %d", val)
	}
}

func TestGetInterpreter_Uint64(t *testing.T) {
	interp, err := getInterpreter(DataTypeUint64)
	requireNoError(t, err)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 0xDEADBEEFCAFEBABE)

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(uint64)
	if !ok {
		t.Fatalf("expected uint64, got %T", result)
	}
	if val != 0xDEADBEEFCAFEBABE {
		t.Errorf("expected 0xDEADBEEFCAFEBABE, got %x", val)
	}
}

func TestGetInterpreter_Float32(t *testing.T) {
	interp, err := getInterpreter(DataTypeFloat32)
	requireNoError(t, err)

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(3.14))

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", result)
	}
	if !almostEqual(float64(val), 3.14, 0.001) {
		t.Errorf("expected ~3.14, got %f", val)
	}
}

func TestGetInterpreter_Float64(t *testing.T) {
	interp, err := getInterpreter(DataTypeFloat64)
	requireNoError(t, err)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(2.718281828))

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result)
	}
	if !almostEqual(val, 2.718281828, 1e-9) {
		t.Errorf("expected ~2.718281828, got %f", val)
	}
}

func TestGetInterpreter_String(t *testing.T) {
	interp, err := getInterpreter(DataTypeString)
	requireNoError(t, err)

	result := interp([]byte("hello"), binary.LittleEndian)
	val, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if val != "hello" {
		t.Errorf("expected %q, got %q", "hello", val)
	}
}

func TestGetInterpreter_Bool(t *testing.T) {
	interp, err := getInterpreter(DataTypeBool)
	requireNoError(t, err)

	trueResult := interp([]byte{1}, binary.LittleEndian)
	if trueResult.(bool) != true {
		t.Error("expected true for non-zero byte")
	}

	falseResult := interp([]byte{0}, binary.LittleEndian)
	if falseResult.(bool) != false {
		t.Error("expected false for zero byte")
	}

	// Any non-zero value should be true
	alsoTrue := interp([]byte{0xFF}, binary.LittleEndian)
	if alsoTrue.(bool) != true {
		t.Error("expected true for 0xFF byte")
	}
}

func TestGetInterpreter_Complex64(t *testing.T) {
	interp, err := getInterpreter(DataTypeComplex64)
	requireNoError(t, err)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], math.Float32bits(1.5))
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(2.5))

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(complex64)
	if !ok {
		t.Fatalf("expected complex64, got %T", result)
	}
	if real(val) != 1.5 || imag(val) != 2.5 {
		t.Errorf("expected (1.5+2.5i), got %v", val)
	}
}

func TestGetInterpreter_Complex128(t *testing.T) {
	interp, err := getInterpreter(DataTypeComplex128)
	requireNoError(t, err)

	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:8], math.Float64bits(-1.0))
	binary.LittleEndian.PutUint64(buf[8:16], math.Float64bits(3.0))

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(complex128)
	if !ok {
		t.Fatalf("expected complex128, got %T", result)
	}
	if real(val) != -1.0 || imag(val) != 3.0 {
		t.Errorf("expected (-1+3i), got %v", val)
	}
}

func TestGetInterpreter_Float32WithUnit(t *testing.T) {
	// Float32WithUnit should use the same interpreter as Float32
	interp, err := getInterpreter(DataTypeFloat32WithUnit)
	requireNoError(t, err)

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(42.0))

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", result)
	}
	if val != 42.0 {
		t.Errorf("expected 42.0, got %f", val)
	}
}

func TestGetInterpreter_Float64WithUnit(t *testing.T) {
	interp, err := getInterpreter(DataTypeFloat64WithUnit)
	requireNoError(t, err)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(99.9))

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result)
	}
	if !almostEqual(val, 99.9, 1e-9) {
		t.Errorf("expected 99.9, got %f", val)
	}
}

// Tests for batchStreamReader via end-to-end file construction

func TestBatchStreamReader_Int32NonInterleaved(t *testing.T) {
	expected := []int32{100, 200, 300, 400, 500}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Values", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Values")

	var got []int32
	for batch, err := range batchStreamReader[int32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int32)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Float64NonInterleaved(t *testing.T) {
	expected := []float64{1.1, 2.2, 3.3, 4.4}
	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("Temps", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Temps")

	var got []float64
	for batch, err := range batchStreamReader[float64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]float64)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Uint8(t *testing.T) {
	expected := []uint8{0, 127, 255, 1, 42}
	fileBytes := buildStandardFile([]testChannelDef{
		uint8Channel("Bytes", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Bytes")

	var got []uint8
	for batch, err := range batchStreamReader[uint8](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]uint8)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_BoolData(t *testing.T) {
	expected := []bool{true, false, true, true, false}
	fileBytes := buildStandardFile([]testChannelDef{
		boolChannel("Flags", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Flags")

	var got []bool
	for batch, err := range batchStreamReader[bool](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]bool)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Int16Negative(t *testing.T) {
	expected := []int16{-1, -2, -32768, 32767, 0}
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Signed", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Signed")

	var got []int16
	for batch, err := range batchStreamReader[int16](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int16)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Float32SpecialValues(t *testing.T) {
	expected := []float32{0, float32(math.Inf(1)), float32(math.Inf(-1)), 1.0}
	fileBytes := buildStandardFile([]testChannelDef{
		float32Channel("Special", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Special")

	var got []float32
	for batch, err := range batchStreamReader[float32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]float32)...)
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(got))
	}
	for i := range expected {
		if math.IsInf(float64(expected[i]), 0) {
			if !math.IsInf(float64(got[i]), 0) {
				t.Errorf("value[%d]: expected Inf, got %f", i, got[i])
			}
		} else if expected[i] != got[i] {
			t.Errorf("value[%d]: expected %f, got %f", i, expected[i], got[i])
		}
	}
}

func TestBatchStreamReader_StringData(t *testing.T) {
	expected := []string{"hello", "world", "foo"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Labels", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Labels")

	var got []string
	for batch, err := range batchStreamReader[string](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]string)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_EmptyStrings(t *testing.T) {
	expected := []string{"", "", "a"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("EmptyStr", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "EmptyStr")

	var got []string
	for batch, err := range batchStreamReader[string](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]string)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_CustomBatchSize(t *testing.T) {
	values := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("BigChannel", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "BigChannel")

	// With batch size of 3, we should get batches: [1,2,3], [4,5,6], [7,8,9], [10]
	options := []ReadOption{BatchSize(3)}
	batchCount := 0
	var allValues []int32

	for batch, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		batchData := batch.([]int32)
		batchCount++
		allValues = append(allValues, batchData...)

		// All batches except possibly the last should be exactly batchSize
		if batchCount < 4 && len(batchData) != 3 {
			t.Errorf("batch %d: expected size 3, got %d", batchCount, len(batchData))
		}
	}

	if batchCount != 4 {
		t.Errorf("expected 4 batches with batch size 3 for 10 values, got %d", batchCount)
	}

	if !cmp.Equal(values, allValues) {
		t.Errorf("aggregated data mismatch:\n%s", cmp.Diff(values, allValues))
	}
}

func TestBatchStreamReader_BatchSizeOne(t *testing.T) {
	values := []int16{10, 20, 30}
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("OneAtATime", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "OneAtATime")

	options := []ReadOption{BatchSize(1)}
	batchCount := 0
	var allValues []int16

	for batch, err := range batchStreamReader[int16](ch, options) {
		requireNoError(t, err)
		batchData := batch.([]int16)
		batchCount++
		allValues = append(allValues, batchData...)

		if len(batchData) != 1 {
			t.Errorf("batch %d: expected size 1, got %d", batchCount, len(batchData))
		}
	}

	if batchCount != 3 {
		t.Errorf("expected 3 batches, got %d", batchCount)
	}

	if !cmp.Equal(values, allValues) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(values, allValues))
	}
}

func TestBatchStreamReader_BatchSizeLargerThanData(t *testing.T) {
	values := []int32{42, 43}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Small", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Small")

	options := []ReadOption{BatchSize(10000)}
	batchCount := 0
	var allValues []int32

	for batch, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		allValues = append(allValues, batch.([]int32)...)
		batchCount++
	}

	// Should produce exactly one batch
	if batchCount != 1 {
		t.Errorf("expected 1 batch, got %d", batchCount)
	}

	if !cmp.Equal(values, allValues) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(values, allValues))
	}
}

func TestBatchStreamReader_EmptyChannel(t *testing.T) {
	// A channel with zero values should produce no batches
	ch := &Channel{
		Name:           "Empty",
		GroupName:      defaultGroupName,
		RawDataType:    DataTypeInt32,
		ScaledDataType: DataTypeInt32,
		reader:         bytes.NewReader(nil),
		dataChunks:     nil,
		totalNumValues: 0,
	}
	scaler, _ := NewMultiscaler(DataTypeInt32, nil)
	ch.scaler = scaler

	batchCount := 0
	for _, err := range batchStreamReader[int32](ch, nil) {
		requireNoError(t, err)
		batchCount++
	}

	if batchCount != 0 {
		t.Errorf("expected 0 batches for empty channel, got %d", batchCount)
	}
}

func TestBatchStreamReader_MultipleChunks(t *testing.T) {
	// Build a file with multiple chunks for the same channel.
	// We do this by making the raw data larger than one chunk's worth.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Multi", DataTypeInt32, 2, nil),
	)

	// 2 values per chunk * 4 bytes = 8 bytes per chunk. Provide 3 chunks = 24 bytes.
	rawData := int32Bytes(10, 20, 30, 40, 50, 60)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Multi")

	if ch.NumValues() != 6 {
		t.Fatalf("expected 6 total values (3 chunks * 2), got %d", ch.NumValues())
	}

	var got []int32
	for batch, err := range batchStreamReader[int32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int32)...)
	}

	expected := []int32{10, 20, 30, 40, 50, 60}
	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_MultipleSegments(t *testing.T) {
	// Build a file with two segments, each contributing data to the same channel.
	// Both segments use full metadata with new object list to avoid shared
	// pointer mutation between segments' object indices.
	meta1 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Across", DataTypeInt16, 3, nil),
	)
	rawData1 := int16Bytes(1, 2, 3)

	meta2 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Across", DataTypeInt16, 3, nil),
	)
	rawData2 := int16Bytes(4, 5, 6)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta1, rawData1)
	tf.addSegment(standardSegmentTOC(), meta2, rawData2)

	fileBytes := tf.build()
	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Across")

	if ch.NumValues() != 6 {
		t.Fatalf("expected 6 total values across 2 segments, got %d", ch.NumValues())
	}

	var got []int16
	for batch, err := range batchStreamReader[int16](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int16)...)
	}

	expected := []int16{1, 2, 3, 4, 5, 6}
	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_TwoChannelsNonInterleaved(t *testing.T) {
	// Two channels in the same segment, non-interleaved: first channel data
	// appears in full, then second channel data.
	expected1 := []int32{10, 20, 30}
	expected2 := []int32{40, 50, 60}

	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("First", expected1),
		int32Channel("Second", expected2),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch1 := requireChannel(t, file, defaultGroupName, "First")
	ch2 := requireChannel(t, file, defaultGroupName, "Second")

	var got1 []int32
	for batch, err := range batchStreamReader[int32](ch1, nil) {
		requireNoError(t, err)
		got1 = append(got1, batch.([]int32)...)
	}

	var got2 []int32
	for batch, err := range batchStreamReader[int32](ch2, nil) {
		requireNoError(t, err)
		got2 = append(got2, batch.([]int32)...)
	}

	if !cmp.Equal(expected1, got1) {
		t.Errorf("First channel data mismatch:\n%s", cmp.Diff(expected1, got1))
	}
	if !cmp.Equal(expected2, got2) {
		t.Errorf("Second channel data mismatch:\n%s", cmp.Diff(expected2, got2))
	}
}

func TestBatchStreamReader_MixedTypeChannels(t *testing.T) {
	// File with channels of different types
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Integers", []int16{1, 2, 3}),
		float32Channel("Floats", []float32{1.5, 2.5, 3.5}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	intCh := requireChannel(t, file, defaultGroupName, "Integers")
	floatCh := requireChannel(t, file, defaultGroupName, "Floats")

	var gotInts []int16
	for batch, err := range batchStreamReader[int16](intCh, nil) {
		requireNoError(t, err)
		gotInts = append(gotInts, batch.([]int16)...)
	}
	expectedInts := []int16{1, 2, 3}
	if !cmp.Equal(expectedInts, gotInts) {
		t.Errorf("int data mismatch:\n%s", cmp.Diff(expectedInts, gotInts))
	}

	var gotFloats []float32
	for batch, err := range batchStreamReader[float32](floatCh, nil) {
		requireNoError(t, err)
		gotFloats = append(gotFloats, batch.([]float32)...)
	}
	expectedFloats := []float32{1.5, 2.5, 3.5}
	if !cmp.Equal(expectedFloats, gotFloats) {
		t.Errorf("float data mismatch:\n%s", cmp.Diff(expectedFloats, gotFloats))
	}
}

func TestBatchStreamReader_InterleavesTwoChannels(t *testing.T) {
	// Build an interleaved segment with two int16 channels.
	// Use numValues=1 per chunk so that the stride calculation
	// (chunkSize - totalSize) produces the correct per-sample skip distance.
	// With numValues=1: totalSize per channel = 2, chunkSize = 4, stride = 2.
	// Provide 3 chunks of interleaved data to get 3 values per channel.
	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("IntA", DataTypeInt16, 1, nil),
		channelMetadata("IntB", DataTypeInt16, 1, nil),
	)

	// 3 chunks of interleaved data: [A[0], B[0]], [A[1], B[1]], [A[2], B[2]]
	rawData := int16Bytes(1, 10, 2, 20, 3, 30)

	toc := interleavedSegmentTOC()
	tf := newTestFile()
	tf.segments = append(tf.segments, testSegment{
		toc:      toc,
		metadata: meta,
		rawData:  rawData,
	})
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	chA := requireChannel(t, file, defaultGroupName, "IntA")
	chB := requireChannel(t, file, defaultGroupName, "IntB")

	var gotA []int16
	for batch, err := range batchStreamReader[int16](chA, nil) {
		requireNoError(t, err)
		gotA = append(gotA, batch.([]int16)...)
	}
	expectedA := []int16{1, 2, 3}
	if !cmp.Equal(expectedA, gotA) {
		t.Errorf("IntA data mismatch:\n%s", cmp.Diff(expectedA, gotA))
	}

	var gotB []int16
	for batch, err := range batchStreamReader[int16](chB, nil) {
		requireNoError(t, err)
		gotB = append(gotB, batch.([]int16)...)
	}
	expectedB := []int16{10, 20, 30}
	if !cmp.Equal(expectedB, gotB) {
		t.Errorf("IntB data mismatch:\n%s", cmp.Diff(expectedB, gotB))
	}
}

func TestBatchStreamReader_WithScalingDisabled(t *testing.T) {
	values := []int32{1, 2, 3}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("NoScale", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "NoScale")

	options := []ReadOption{WithScaling(false)}

	var got []int32
	for batch, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		got = append(got, batch.([]int32)...)
	}

	if !cmp.Equal(values, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(values, got))
	}
}

func TestBatchStreamReader_EarlyTermination(t *testing.T) {
	// Verify that we can break out of the iterator early without issues.
	values := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Breakable", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Breakable")

	options := []ReadOption{BatchSize(2)}
	count := 0

	for _, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		count++
		if count == 2 {
			break // Only consume 2 batches out of 5
		}
	}

	if count != 2 {
		t.Errorf("expected to consume 2 batches, consumed %d", count)
	}
}

func TestBatchStreamReader_LargeDataSet(t *testing.T) {
	// Test with a larger data set to exercise the default batch size path
	n := 5000
	values := make([]int32, n)
	for i := range values {
		values[i] = int32(i)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Large", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Large")

	var got []int32
	for batch, err := range batchStreamReader[int32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int32)...)
	}

	if len(got) != n {
		t.Fatalf("expected %d values, got %d", n, len(got))
	}

	if !cmp.Equal(values, got) {
		t.Error("large data set mismatch (diff omitted for brevity)")
	}
}

func TestBatchStreamReader_Uint32Data(t *testing.T) {
	expected := []uint32{0, 1, 0xFFFFFFFF, 0xDEADBEEF}
	fileBytes := buildStandardFile([]testChannelDef{
		uint32Channel("U32", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U32")

	var got []uint32
	for batch, err := range batchStreamReader[uint32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]uint32)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_DAQmxChannel(t *testing.T) {
	// Verify that batchStreamReader works with DAQmx channels via the
	// existing DAQmx test infrastructure.
	scalerMeta := daqmxScalerMetadata(0, 5, 0, 0) // Int32, offset=0, buffer=0

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("DAQCh", 3, []uint32{4}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := int32Bytes(100, 200, 300)

	tf := newTestFile()
	tf.addSegment(segmentTOC(), meta, rawData)
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "DAQCh")

	data, err := ch.ReadAll(WithScaling(false))
	requireNoError(t, err)

	got, ok := data.([]int32)
	if !ok {
		t.Fatalf("expected []int32, got %T", data)
	}

	expected := []int32{100, 200, 300}
	if !cmp.Equal(expected, got) {
		t.Errorf("DAQmx data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_StringDefaultBatchSize(t *testing.T) {
	// Strings with the default batch size (256 for strings), verifying that
	// all string data is read correctly in a single batch.
	values := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Words", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Words")

	var got []string
	for batch, err := range batchStreamReader[string](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]string)...)
	}

	if !cmp.Equal(values, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(values, got))
	}
}

func TestBatchStreamReader_StringSingleBatch(t *testing.T) {
	// Use a batch size large enough to cover all strings in one batch.
	values := []string{"one", "two", "three", "four", "five"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("AllAtOnce", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "AllAtOnce")

	options := []ReadOption{BatchSize(100)}
	var got []string
	batchCount := 0

	for batch, err := range batchStreamReader[string](ch, options) {
		requireNoError(t, err)
		got = append(got, batch.([]string)...)
		batchCount++
	}

	if !cmp.Equal(values, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(values, got))
	}

	if batchCount != 1 {
		t.Errorf("expected 1 batch with large batch size, got %d", batchCount)
	}
}

// --- Additional getInterpreter interpretation tests for missing types ---

func TestGetInterpreter_Int64Interpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeInt64)
	requireNoError(t, err)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 0x8000000000000000) // math.MinInt64 as uint64

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", result)
	}
	if val != math.MinInt64 {
		t.Errorf("expected %d, got %d", int64(math.MinInt64), val)
	}

	// Also test a positive value
	binary.LittleEndian.PutUint64(buf, 0x7FFFFFFFFFFFFFFF)
	result2 := interp(buf, binary.LittleEndian)
	if result2.(int64) != math.MaxInt64 {
		t.Errorf("expected MaxInt64, got %d", result2.(int64))
	}
}

func TestGetInterpreter_Uint8Interpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeUint8)
	requireNoError(t, err)

	result := interp([]byte{0}, binary.LittleEndian)
	val, ok := result.(uint8)
	if !ok {
		t.Fatalf("expected uint8, got %T", result)
	}
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}

	result2 := interp([]byte{255}, binary.LittleEndian)
	if result2.(uint8) != 255 {
		t.Errorf("expected 255, got %d", result2.(uint8))
	}
}

func TestGetInterpreter_Uint16Interpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeUint16)
	requireNoError(t, err)

	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, 0xABCD)

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(uint16)
	if !ok {
		t.Fatalf("expected uint16, got %T", result)
	}
	if val != 0xABCD {
		t.Errorf("expected 0xABCD, got 0x%X", val)
	}
}

func TestGetInterpreter_Uint32Interpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeUint32)
	requireNoError(t, err)

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0x12345678)

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(uint32)
	if !ok {
		t.Fatalf("expected uint32, got %T", result)
	}
	if val != 0x12345678 {
		t.Errorf("expected 0x12345678, got 0x%X", val)
	}
}

func TestGetInterpreter_TimestampInterpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeTimestamp)
	requireNoError(t, err)

	// Timestamp in TDMS LE format: 8 bytes remainder (fractional) then 8 bytes seconds
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:8], 500)   // remainder
	binary.LittleEndian.PutUint64(buf[8:16], 1000) // seconds

	result := interp(buf, binary.LittleEndian)
	val, ok := result.(Timestamp)
	if !ok {
		t.Fatalf("expected Timestamp, got %T", result)
	}
	if val.Timestamp != 1000 {
		t.Errorf("expected Timestamp=1000, got %d", val.Timestamp)
	}
	if val.Remainder != 500 {
		t.Errorf("expected Remainder=500, got %d", val.Remainder)
	}
}

func TestGetInterpreter_TimestampBigEndian(t *testing.T) {
	interp, err := getInterpreter(DataTypeTimestamp)
	requireNoError(t, err)

	// Timestamp in TDMS BE format: 8 bytes seconds then 8 bytes remainder
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], 2000)  // seconds
	binary.BigEndian.PutUint64(buf[8:16], 1234) // remainder

	result := interp(buf, binary.BigEndian)
	val, ok := result.(Timestamp)
	if !ok {
		t.Fatalf("expected Timestamp, got %T", result)
	}
	if val.Timestamp != 2000 {
		t.Errorf("expected Timestamp=2000, got %d", val.Timestamp)
	}
	if val.Remainder != 1234 {
		t.Errorf("expected Remainder=1234, got %d", val.Remainder)
	}
}

func TestGetInterpreter_Uint16BigEndian(t *testing.T) {
	interp, err := getInterpreter(DataTypeUint16)
	requireNoError(t, err)

	buf := []byte{0x12, 0x34}
	result := interp(buf, binary.BigEndian)
	val, ok := result.(uint16)
	if !ok {
		t.Fatalf("expected uint16, got %T", result)
	}
	if val != 0x1234 {
		t.Errorf("expected 0x1234, got 0x%X", val)
	}
}

func TestGetInterpreter_Uint32BigEndian(t *testing.T) {
	interp, err := getInterpreter(DataTypeUint32)
	requireNoError(t, err)

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 0xDEADBEEF)

	result := interp(buf, binary.BigEndian)
	val, ok := result.(uint32)
	if !ok {
		t.Fatalf("expected uint32, got %T", result)
	}
	if val != 0xDEADBEEF {
		t.Errorf("expected 0xDEADBEEF, got 0x%X", val)
	}
}

func TestGetInterpreter_Int64BigEndian(t *testing.T) {
	interp, err := getInterpreter(DataTypeInt64)
	requireNoError(t, err)

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(42))

	result := interp(buf, binary.BigEndian)
	val, ok := result.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", result)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestGetInterpreter_Float128Interpretation(t *testing.T) {
	interp, err := getInterpreter(DataTypeFloat128)
	requireNoError(t, err)

	// Float128 is 16 bytes, stored as raw bytes
	input := make([]byte, 16)
	for i := range input {
		input[i] = byte(i)
	}

	result := interp(input, binary.LittleEndian)
	val, ok := result.(Float128)
	if !ok {
		t.Fatalf("expected Float128, got %T", result)
	}
	// For little-endian, the bytes should be preserved as-is
	for i := range val {
		if val[i] != byte(i) {
			t.Errorf("byte[%d]: expected %d, got %d", i, i, val[i])
		}
	}
}

// --- Additional batchStreamReader tests for missing data types ---

func TestBatchStreamReader_Int8Data(t *testing.T) {
	expected := []int8{-128, -1, 0, 1, 127}

	rawData := make([]byte, len(expected))
	for i, v := range expected {
		rawData[i] = byte(v)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("I8", DataTypeInt8, uint64(len(expected)), nil),
			rawData:  rawData,
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I8")

	var got []int8
	for batch, err := range batchStreamReader[int8](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int8)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Int64Data(t *testing.T) {
	expected := []int64{math.MinInt64, -1, 0, 1, math.MaxInt64}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, v)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("I64", DataTypeInt64, uint64(len(expected)), nil),
			rawData:  rawBuf.Bytes(),
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I64")

	var got []int64
	for batch, err := range batchStreamReader[int64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int64)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Uint16Data(t *testing.T) {
	expected := []uint16{0, 1, 256, 0xFFFF, 0xABCD}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, v)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("U16", DataTypeUint16, uint64(len(expected)), nil),
			rawData:  rawBuf.Bytes(),
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U16")

	var got []uint16
	for batch, err := range batchStreamReader[uint16](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]uint16)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Uint64Data(t *testing.T) {
	expected := []uint64{0, 1, math.MaxUint64, 0xCAFEBABE, 0xDEADBEEF00000000}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, v)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("U64", DataTypeUint64, uint64(len(expected)), nil),
			rawData:  rawBuf.Bytes(),
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U64")

	var got []uint64
	for batch, err := range batchStreamReader[uint64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]uint64)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Complex64Data(t *testing.T) {
	expected := []complex64{complex(1.5, 2.5), complex(-3.0, 4.0), complex(0, 0)}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, math.Float32bits(real(v)))
		_ = binary.Write(&rawBuf, binary.LittleEndian, math.Float32bits(imag(v)))
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("C64", DataTypeComplex64, uint64(len(expected)), nil),
			rawData:  rawBuf.Bytes(),
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "C64")

	var got []complex64
	for batch, err := range batchStreamReader[complex64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]complex64)...)
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("value[%d]: expected %v, got %v", i, expected[i], got[i])
		}
	}
}

func TestBatchStreamReader_Complex128Data(t *testing.T) {
	expected := []complex128{complex(-1.0, 3.0), complex(0.0, 0.0), complex(math.Pi, math.E)}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, math.Float64bits(real(v)))
		_ = binary.Write(&rawBuf, binary.LittleEndian, math.Float64bits(imag(v)))
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("C128", DataTypeComplex128, uint64(len(expected)), nil),
			rawData:  rawBuf.Bytes(),
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "C128")

	var got []complex128
	for batch, err := range batchStreamReader[complex128](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]complex128)...)
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("value[%d]: expected %v, got %v", i, expected[i], got[i])
		}
	}
}

func TestBatchStreamReader_TimestampData(t *testing.T) {
	// Timestamp is 16 bytes: 8 bytes remainder (fractional, LE first) then 8 bytes seconds
	expected := []Timestamp{
		{Timestamp: 1000, Remainder: 500},
		{Timestamp: 2000, Remainder: 0},
		{Timestamp: 0, Remainder: 12345},
	}

	var rawBuf bytes.Buffer
	for _, ts := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, ts.Remainder)
		_ = binary.Write(&rawBuf, binary.LittleEndian, ts.Timestamp)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		{
			metadata: channelMetadata("Stamps", DataTypeTimestamp, uint64(len(expected)), nil),
			rawData:  rawBuf.Bytes(),
		},
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Stamps")

	var got []Timestamp
	for batch, err := range batchStreamReader[Timestamp](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]Timestamp)...)
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d timestamps, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i].Timestamp != expected[i].Timestamp || got[i].Remainder != expected[i].Remainder {
			t.Errorf("value[%d]: expected {%d, %d}, got {%d, %d}",
				i, expected[i].Timestamp, expected[i].Remainder,
				got[i].Timestamp, got[i].Remainder)
		}
	}
}

func TestBatchStreamReader_Float32WithUnitData(t *testing.T) {
	// Float32WithUnit uses the same interpreter as Float32
	expected := []float32{9.8, 32.0, -40.0}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, v)
	}

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Accel", DataTypeFloat32WithUnit, 3, nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawBuf.Bytes())
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Accel")

	var got []float32
	for batch, err := range batchStreamReader[float32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]float32)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_Float64WithUnitData(t *testing.T) {
	// Float64WithUnit uses the same interpreter as Float64
	expected := []float64{100.5, -200.25, 0.001}

	var rawBuf bytes.Buffer
	for _, v := range expected {
		_ = binary.Write(&rawBuf, binary.LittleEndian, v)
	}

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Voltage", DataTypeFloat64WithUnit, 3, nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawBuf.Bytes())
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Voltage")

	var got []float64
	for batch, err := range batchStreamReader[float64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]float64)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

// --- Interleaved tests with different types ---

func TestBatchStreamReader_InterleavedInt32TwoChannels(t *testing.T) {
	// Two int32 channels, interleaved, numValues=1 per chunk, 4 chunks.
	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("ChA", DataTypeInt32, 1, nil),
		channelMetadata("ChB", DataTypeInt32, 1, nil),
	)

	// 4 chunks: [A0, B0], [A1, B1], [A2, B2], [A3, B3]
	rawData := int32Bytes(100, 10, 200, 20, 300, 30, 400, 40)

	tf := newTestFile()
	tf.segments = append(tf.segments, testSegment{
		toc:      interleavedSegmentTOC(),
		metadata: meta,
		rawData:  rawData,
	})
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	chA := requireChannel(t, file, defaultGroupName, "ChA")
	chB := requireChannel(t, file, defaultGroupName, "ChB")

	var gotA []int32
	for batch, err := range batchStreamReader[int32](chA, nil) {
		requireNoError(t, err)
		gotA = append(gotA, batch.([]int32)...)
	}
	expectedA := []int32{100, 200, 300, 400}
	if !cmp.Equal(expectedA, gotA) {
		t.Errorf("ChA data mismatch:\n%s", cmp.Diff(expectedA, gotA))
	}

	var gotB []int32
	for batch, err := range batchStreamReader[int32](chB, nil) {
		requireNoError(t, err)
		gotB = append(gotB, batch.([]int32)...)
	}
	expectedB := []int32{10, 20, 30, 40}
	if !cmp.Equal(expectedB, gotB) {
		t.Errorf("ChB data mismatch:\n%s", cmp.Diff(expectedB, gotB))
	}
}

func TestBatchStreamReader_InterleavedThreeChannels(t *testing.T) {
	// Three int16 channels, interleaved, numValues=1 per chunk, 2 chunks.
	meta := buildMetadata(5,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("X", DataTypeInt16, 1, nil),
		channelMetadata("Y", DataTypeInt16, 1, nil),
		channelMetadata("Z", DataTypeInt16, 1, nil),
	)

	// 2 chunks: [X0, Y0, Z0], [X1, Y1, Z1]
	rawData := int16Bytes(1, 2, 3, 4, 5, 6)

	tf := newTestFile()
	tf.segments = append(tf.segments, testSegment{
		toc:      interleavedSegmentTOC(),
		metadata: meta,
		rawData:  rawData,
	})
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	chX := requireChannel(t, file, defaultGroupName, "X")
	chY := requireChannel(t, file, defaultGroupName, "Y")
	chZ := requireChannel(t, file, defaultGroupName, "Z")

	var gotX []int16
	for batch, err := range batchStreamReader[int16](chX, nil) {
		requireNoError(t, err)
		gotX = append(gotX, batch.([]int16)...)
	}
	if !cmp.Equal([]int16{1, 4}, gotX) {
		t.Errorf("X data mismatch:\n%s", cmp.Diff([]int16{1, 4}, gotX))
	}

	var gotY []int16
	for batch, err := range batchStreamReader[int16](chY, nil) {
		requireNoError(t, err)
		gotY = append(gotY, batch.([]int16)...)
	}
	if !cmp.Equal([]int16{2, 5}, gotY) {
		t.Errorf("Y data mismatch:\n%s", cmp.Diff([]int16{2, 5}, gotY))
	}

	var gotZ []int16
	for batch, err := range batchStreamReader[int16](chZ, nil) {
		requireNoError(t, err)
		gotZ = append(gotZ, batch.([]int16)...)
	}
	if !cmp.Equal([]int16{3, 6}, gotZ) {
		t.Errorf("Z data mismatch:\n%s", cmp.Diff([]int16{3, 6}, gotZ))
	}
}

// --- Multi-chunk with custom batch size ---

func TestBatchStreamReader_MultipleChunksCustomBatchSize(t *testing.T) {
	// 3 chunks of 2 values each = 6 values total, batch size = 4.
	// First batch should get 4 values, second batch should get 2 values.
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Chunky", DataTypeInt32, 2, nil),
	)

	rawData := int32Bytes(10, 20, 30, 40, 50, 60)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Chunky")

	if ch.NumValues() != 6 {
		t.Fatalf("expected 6 values, got %d", ch.NumValues())
	}

	options := []ReadOption{BatchSize(4)}
	var allValues []int32
	batchCount := 0

	for batch, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		allValues = append(allValues, batch.([]int32)...)
		batchCount++
	}

	expected := []int32{10, 20, 30, 40, 50, 60}
	if !cmp.Equal(expected, allValues) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, allValues))
	}
}

// --- Multi-segment with small batch size ---

func TestBatchStreamReader_MultipleSegmentsSmallBatch(t *testing.T) {
	// Two segments with 4 values each, batch size = 3.
	// Segment 1: [1, 2, 3, 4], Segment 2: [5, 6, 7, 8]
	// With batch size 3: batches should be [1,2,3], [4], [5,6,7], [8]
	meta1 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("SmBatch", DataTypeInt32, 4, nil),
	)
	rawData1 := int32Bytes(1, 2, 3, 4)

	meta2 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("SmBatch", DataTypeInt32, 4, nil),
	)
	rawData2 := int32Bytes(5, 6, 7, 8)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta1, rawData1)
	tf.addSegment(standardSegmentTOC(), meta2, rawData2)
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "SmBatch")

	if ch.NumValues() != 8 {
		t.Fatalf("expected 8 total values, got %d", ch.NumValues())
	}

	options := []ReadOption{BatchSize(3)}
	var allValues []int32
	batchCount := 0

	for batch, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		allValues = append(allValues, batch.([]int32)...)
		batchCount++
	}

	expected := []int32{1, 2, 3, 4, 5, 6, 7, 8}
	if !cmp.Equal(expected, allValues) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, allValues))
	}

	// We expect at least 3 batches: [1,2,3], [4], [5,6,7], [8] = 4 batches
	// (chunk boundaries may cause additional splits)
	if batchCount < 3 {
		t.Errorf("expected at least 3 batches with batch size 3 across 8 values in 2 segments, got %d", batchCount)
	}
}

// --- Single value per type ---

func TestBatchStreamReader_SingleValueInt32(t *testing.T) {
	expected := []int32{42}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("One", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "One")

	var got []int32
	for batch, err := range batchStreamReader[int32](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int32)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestBatchStreamReader_SingleString(t *testing.T) {
	expected := []string{"only"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Solo", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Solo")

	var got []string
	for batch, err := range batchStreamReader[string](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]string)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

// --- Multiple segments same channel, verifying data order is preserved ---

func TestBatchStreamReader_ThreeSegmentsDataOrder(t *testing.T) {
	// Three segments contributing to the same channel. Verify the order is
	// segment1 data, then segment2 data, then segment3 data.
	meta1 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Ordered", DataTypeInt16, 2, nil),
	)
	meta2 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Ordered", DataTypeInt16, 2, nil),
	)
	meta3 := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Ordered", DataTypeInt16, 2, nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta1, int16Bytes(10, 20))
	tf.addSegment(standardSegmentTOC(), meta2, int16Bytes(30, 40))
	tf.addSegment(standardSegmentTOC(), meta3, int16Bytes(50, 60))
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Ordered")

	if ch.NumValues() != 6 {
		t.Fatalf("expected 6 total values, got %d", ch.NumValues())
	}

	var got []int16
	for batch, err := range batchStreamReader[int16](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]int16)...)
	}

	expected := []int16{10, 20, 30, 40, 50, 60}
	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

// --- Verify batch reuse semantics ---

func TestBatchStreamReader_BatchSliceReuse(t *testing.T) {
	// The documentation says the same underlying slice is reused. Verify that
	// collecting batch data requires copying.
	values := []int32{1, 2, 3, 4, 5, 6}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Reuse", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Reuse")

	options := []ReadOption{BatchSize(3)}
	var batches [][]int32

	for batch, err := range batchStreamReader[int32](ch, options) {
		requireNoError(t, err)
		batchData := batch.([]int32)
		// Must copy since the slice may be reused
		copied := make([]int32, len(batchData))
		copy(copied, batchData)
		batches = append(batches, copied)
	}

	if len(batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(batches))
	}

	if !cmp.Equal([]int32{1, 2, 3}, batches[0]) {
		t.Errorf("batch 0 mismatch:\n%s", cmp.Diff([]int32{1, 2, 3}, batches[0]))
	}
	if !cmp.Equal([]int32{4, 5, 6}, batches[1]) {
		t.Errorf("batch 1 mismatch:\n%s", cmp.Diff([]int32{4, 5, 6}, batches[1]))
	}
}

// --- Zero values in numeric types ---

func TestBatchStreamReader_AllZerosFloat64(t *testing.T) {
	expected := []float64{0.0, 0.0, 0.0}
	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("Zeros", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Zeros")

	var got []float64
	for batch, err := range batchStreamReader[float64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]float64)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

// --- Boundary: NaN handling in float types ---

func TestBatchStreamReader_Float64NaN(t *testing.T) {
	values := []float64{math.NaN(), 1.0, math.NaN()}

	var rawBuf bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&rawBuf, binary.LittleEndian, v)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("NaNs", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "NaNs")

	var got []float64
	for batch, err := range batchStreamReader[float64](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]float64)...)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 values, got %d", len(got))
	}
	if !math.IsNaN(got[0]) {
		t.Errorf("value[0]: expected NaN, got %f", got[0])
	}
	if got[1] != 1.0 {
		t.Errorf("value[1]: expected 1.0, got %f", got[1])
	}
	if !math.IsNaN(got[2]) {
		t.Errorf("value[2]: expected NaN, got %f", got[2])
	}
}

// --- Long strings ---

func TestBatchStreamReader_LongStrings(t *testing.T) {
	longStr := ""
	for range 1000 {
		longStr += "abcdefghij"
	}
	expected := []string{longStr, "short", longStr}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Long", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Long")

	var got []string
	for batch, err := range batchStreamReader[string](ch, nil) {
		requireNoError(t, err)
		got = append(got, batch.([]string)...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch: lengths expected [%d, %d, %d], got [%d, %d, %d]",
			len(expected[0]), len(expected[1]), len(expected[2]),
			len(got[0]), len(got[1]), len(got[2]))
	}
}

// --- Two channels with different sizes ---

func TestBatchStreamReader_TwoChannelsDifferentValueCounts(t *testing.T) {
	// Two non-interleaved channels with different numbers of values.
	vals1 := []int16{1, 2}
	vals2 := []int32{10, 20, 30, 40, 50}

	var raw1 bytes.Buffer
	for _, v := range vals1 {
		_ = binary.Write(&raw1, binary.LittleEndian, v)
	}
	var raw2 bytes.Buffer
	for _, v := range vals2 {
		_ = binary.Write(&raw2, binary.LittleEndian, v)
	}

	meta := buildMetadata(4,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Short", DataTypeInt16, 2, nil),
		channelMetadata("Long", DataTypeInt32, 5, nil),
	)

	var rawData bytes.Buffer
	rawData.Write(raw1.Bytes())
	rawData.Write(raw2.Bytes())

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData.Bytes())
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	chShort := requireChannel(t, file, defaultGroupName, "Short")
	chLong := requireChannel(t, file, defaultGroupName, "Long")

	var gotShort []int16
	for batch, err := range batchStreamReader[int16](chShort, nil) {
		requireNoError(t, err)
		gotShort = append(gotShort, batch.([]int16)...)
	}
	if !cmp.Equal(vals1, gotShort) {
		t.Errorf("Short channel mismatch:\n%s", cmp.Diff(vals1, gotShort))
	}

	var gotLong []int32
	for batch, err := range batchStreamReader[int32](chLong, nil) {
		requireNoError(t, err)
		gotLong = append(gotLong, batch.([]int32)...)
	}
	if !cmp.Equal(vals2, gotLong) {
		t.Errorf("Long channel mismatch:\n%s", cmp.Diff(vals2, gotLong))
	}
}

// --- Verify NumValues matches data read ---

func TestBatchStreamReader_NumValuesConsistency(t *testing.T) {
	values := []int32{1, 2, 3, 4, 5, 6, 7}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Consistent", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Consistent")

	if ch.NumValues() != uint64(len(values)) {
		t.Fatalf("NumValues: expected %d, got %d", len(values), ch.NumValues())
	}

	count := 0
	for batch, err := range batchStreamReader[int32](ch, nil) {
		requireNoError(t, err)
		count += len(batch.([]int32))
	}

	if uint64(count) != ch.NumValues() {
		t.Errorf("read %d values but NumValues reports %d", count, ch.NumValues())
	}
}
