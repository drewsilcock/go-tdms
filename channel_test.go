package tdms

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestChannel_NumValues(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Counts", []int32{10, 20, 30, 40, 50}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Counts")

	if ch.NumValues() != 5 {
		t.Errorf("expected NumValues()=5, got %d", ch.NumValues())
	}
}

func TestChannel_NumValuesZero(t *testing.T) {
	// Construct a Channel directly with no data chunks to verify NumValues
	// returns 0. We avoid parsing a no-data channel through the full file
	// path because the library panics on segments with chunkSize=0.
	ch := &Channel{
		Name:           "NoData",
		GroupName:      defaultGroupName,
		RawDataType:    DataTypeInt32,
		ScaledDataType: DataTypeInt32,
		reader:         bytes.NewReader(nil),
		dataChunks:     nil,
		totalNumValues: 0,
	}

	if ch.NumValues() != 0 {
		t.Errorf("expected NumValues()=0, got %d", ch.NumValues())
	}
}

func TestChannel_NumValuesAcrossMultipleSegments(t *testing.T) {
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Multi", DataTypeInt16, 3, nil),
	)
	raw1 := int16Bytes(1, 2, 3)
	raw2 := int16Bytes(4, 5, 6)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, raw1)
	tf.addSegment(standardSegmentTOC(), meta, raw2)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Multi")

	if ch.NumValues() != 6 {
		t.Errorf("expected NumValues()=6 across 2 segments, got %d", ch.NumValues())
	}
}

func TestChannel_NumValuesAcrossMultipleChunks(t *testing.T) {
	// 2 values per chunk, 6 values total = 3 chunks
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
		t.Errorf("expected NumValues()=6, got %d", ch.NumValues())
	}
}

func TestRenderReadOptions_Defaults(t *testing.T) {
	opts := renderReadOptions(nil)

	if opts.batchSize != 0 {
		t.Errorf("expected default batchSize=0, got %d", opts.batchSize)
	}
	if !opts.shouldScale {
		t.Error("expected default shouldScale=true")
	}
	if opts.daqmxScaleIndex != 0 {
		t.Errorf("expected default daqmxScaleIndex=0, got %d", opts.daqmxScaleIndex)
	}
}

func TestRenderReadOptions_CustomBatchSize(t *testing.T) {
	opts := renderReadOptions([]ReadOption{BatchSize(512)})

	if opts.batchSize != 512 {
		t.Errorf("expected batchSize=512, got %d", opts.batchSize)
	}
}

func TestRenderReadOptions_WithScalingTrue(t *testing.T) {
	opts := renderReadOptions([]ReadOption{WithScaling(true)})

	if !opts.shouldScale {
		t.Error("expected shouldScale=true")
	}
}

func TestRenderReadOptions_WithScalingFalse(t *testing.T) {
	opts := renderReadOptions([]ReadOption{WithScaling(false)})

	if opts.shouldScale {
		t.Error("expected shouldScale=false")
	}
}

func TestRenderReadOptions_WithScalingNoArg(t *testing.T) {
	// WithScaling() with no args should default to true
	opts := renderReadOptions([]ReadOption{WithScaling()})

	if !opts.shouldScale {
		t.Error("expected shouldScale=true when WithScaling() called with no args")
	}
}

func TestRenderReadOptions_ForDAQmxScaler(t *testing.T) {
	opts := renderReadOptions([]ReadOption{ForDAQmxScaler(3)})

	if opts.daqmxScaleIndex != 3 {
		t.Errorf("expected daqmxScaleIndex=3, got %d", opts.daqmxScaleIndex)
	}
}

func TestRenderReadOptions_MultipleOptions(t *testing.T) {
	opts := renderReadOptions([]ReadOption{
		BatchSize(1024),
		WithScaling(false),
		ForDAQmxScaler(2),
	})

	if opts.batchSize != 1024 {
		t.Errorf("expected batchSize=1024, got %d", opts.batchSize)
	}
	if opts.shouldScale {
		t.Error("expected shouldScale=false")
	}
	if opts.daqmxScaleIndex != 2 {
		t.Errorf("expected daqmxScaleIndex=2, got %d", opts.daqmxScaleIndex)
	}
}

func TestRenderReadOptions_LastOptionWins(t *testing.T) {
	opts := renderReadOptions([]ReadOption{
		BatchSize(100),
		BatchSize(200),
		BatchSize(300),
	})

	if opts.batchSize != 300 {
		t.Errorf("expected last batchSize=300 to win, got %d", opts.batchSize)
	}
}

func TestChannel_DataType_NonDAQmx(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("I32", []int32{1}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I32")

	dt, err := ch.DataType()
	requireNoError(t, err)

	if dt != DataTypeInt32 {
		t.Errorf("expected DataTypeInt32, got %s", dt)
	}
}

func TestChannel_DataType_ScalingDisabled(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Raw", []int16{1}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Raw")

	dt, err := ch.DataType(WithScaling(false))
	requireNoError(t, err)

	if dt != DataTypeInt16 {
		t.Errorf("expected DataTypeInt16 with scaling disabled, got %s", dt)
	}
}

func TestChannel_ReadInt16All(t *testing.T) {
	expected := []int16{-100, 0, 100, 32767, -32768}
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("I16", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I16")

	data, err := ch.ReadInt16All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadInt32All(t *testing.T) {
	expected := []int32{-1, 0, 1, 2147483647, -2147483648}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("I32", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I32")

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadUint8All(t *testing.T) {
	expected := []uint8{0, 1, 127, 255}
	fileBytes := buildStandardFile([]testChannelDef{
		uint8Channel("U8", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U8")

	data, err := ch.ReadUint8All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadUint32All(t *testing.T) {
	expected := []uint32{0, 1, 0xFFFFFFFF, 0xDEADBEEF}
	fileBytes := buildStandardFile([]testChannelDef{
		uint32Channel("U32", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U32")

	data, err := ch.ReadUint32All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadFloat32All(t *testing.T) {
	expected := []float32{0.0, 1.5, -3.14, float32(math.Inf(1))}
	fileBytes := buildStandardFile([]testChannelDef{
		float32Channel("F32", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "F32")

	data, err := ch.ReadFloat32All()
	requireNoError(t, err)

	if len(data) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(data))
	}
	for i := range expected {
		if math.IsInf(float64(expected[i]), 0) {
			if !math.IsInf(float64(data[i]), 0) {
				t.Errorf("[%d]: expected Inf, got %f", i, data[i])
			}
		} else if !almostEqual(float64(expected[i]), float64(data[i]), 1e-5) {
			t.Errorf("[%d]: expected %f, got %f", i, expected[i], data[i])
		}
	}
}

func TestChannel_ReadFloat64All(t *testing.T) {
	expected := []float64{0.0, 2.718281828, -1.41421356, math.Pi}
	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("F64", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "F64")

	data, err := ch.ReadFloat64All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadBoolAll(t *testing.T) {
	expected := []bool{true, false, false, true, true}
	fileBytes := buildStandardFile([]testChannelDef{
		boolChannel("Flags", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Flags")

	data, err := ch.ReadBoolAll()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadStringAll(t *testing.T) {
	expected := []string{"hello", "world", "test", "data"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Labels", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Labels")

	data, err := ch.ReadStringAll()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadStringAll_EmptyStrings(t *testing.T) {
	expected := []string{"", "a", "", "bc", ""}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Mixed", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Mixed")

	data, err := ch.ReadStringAll()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadStringAll_LongStrings(t *testing.T) {
	long := ""
	for range 500 {
		long += "x"
	}
	expected := []string{long, "short", long}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("Long", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Long")

	data, err := ch.ReadStringAll()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadAll_DynamicTyping(t *testing.T) {
	tests := []struct {
		name     string
		channel  testChannelDef
		expected any
	}{
		{
			name:     "Int16",
			channel:  int16Channel("Int16", []int16{1, 2, 3}),
			expected: []int16{1, 2, 3},
		},
		{
			name:     "Int32",
			channel:  int32Channel("Int32", []int32{10, 20}),
			expected: []int32{10, 20},
		},
		{
			name:     "Uint8",
			channel:  uint8Channel("Uint8", []uint8{0, 255}),
			expected: []uint8{0, 255},
		},
		{
			name:     "Uint32",
			channel:  uint32Channel("Uint32", []uint32{42}),
			expected: []uint32{42},
		},
		{
			name:     "Float64",
			channel:  float64Channel("Float64", []float64{3.14}),
			expected: []float64{3.14},
		},
		{
			name:     "Bool",
			channel:  boolChannel("Bool", []bool{true}),
			expected: []bool{true},
		},
		{
			name:     "String",
			channel:  stringChannel("String", []string{"abc"}),
			expected: []string{"abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileBytes := buildStandardFile([]testChannelDef{tt.channel})

			file, err := readTDMSFromBytes(fileBytes)
			requireNoError(t, err)

			// The channel name is determined by the test channel def which uses
			// the same name as the test name.
			ch := requireChannel(t, file, defaultGroupName, tt.name)

			data, err := ch.ReadAll()
			requireNoError(t, err)

			if !cmp.Equal(tt.expected, data) {
				t.Errorf("data mismatch:\n%s", cmp.Diff(tt.expected, data))
			}
		})
	}
}

func TestChannel_Read_StreamsIndividualValues(t *testing.T) {
	expected := []int32{10, 20, 30, 40}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Stream", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Stream")

	var got []int32
	for val, err := range ch.ReadInt32() {
		requireNoError(t, err)
		got = append(got, val)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("streaming data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_Read_EarlyBreak(t *testing.T) {
	values := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Break", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Break")

	count := 0
	for _, err := range ch.ReadInt32() {
		requireNoError(t, err)
		count++
		if count == 3 {
			break
		}
	}

	if count != 3 {
		t.Errorf("expected to break after 3 values, stopped at %d", count)
	}
}

func TestChannel_ReadFloat64_Stream(t *testing.T) {
	expected := []float64{1.1, 2.2, 3.3}
	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("FStream", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "FStream")

	var got []float64
	for val, err := range ch.ReadFloat64() {
		requireNoError(t, err)
		got = append(got, val)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadString_Stream(t *testing.T) {
	expected := []string{"one", "two", "three"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("SStream", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "SStream")

	var got []string
	for val, err := range ch.ReadString() {
		requireNoError(t, err)
		got = append(got, val)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadInt16Batch(t *testing.T) {
	expected := []int16{1, 2, 3, 4, 5}
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("BatchCh", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "BatchCh")

	var got []int16
	batchCount := 0
	for batch, err := range ch.ReadInt16Batch(BatchSize(2)) {
		requireNoError(t, err)
		got = append(got, batch...)
		batchCount++
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}

	// 5 values with batch size 2: 3 batches (2, 2, 1)
	if batchCount != 3 {
		t.Errorf("expected 3 batches, got %d", batchCount)
	}
}

func TestChannel_ReadInt32Batch(t *testing.T) {
	expected := []int32{100, 200, 300}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("I32Batch", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I32Batch")

	var got []int32
	for batch, err := range ch.ReadInt32Batch(BatchSize(10)) {
		requireNoError(t, err)
		got = append(got, batch...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadFloat64Batch(t *testing.T) {
	expected := []float64{1.1, 2.2, 3.3, 4.4, 5.5, 6.6}
	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("F64Batch", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "F64Batch")

	var got []float64
	batchCount := 0
	for batch, err := range ch.ReadFloat64Batch(BatchSize(4)) {
		requireNoError(t, err)
		got = append(got, batch...)
		batchCount++
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}

	if batchCount != 2 {
		t.Errorf("expected 2 batches (4+2), got %d", batchCount)
	}
}

func TestChannel_ReadBoolBatch(t *testing.T) {
	expected := []bool{true, false, true}
	fileBytes := buildStandardFile([]testChannelDef{
		boolChannel("BoolBatch", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "BoolBatch")

	var got []bool
	for batch, err := range ch.ReadBoolBatch(BatchSize(2)) {
		requireNoError(t, err)
		got = append(got, batch...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadBatch_GenericInterface(t *testing.T) {
	expected := []int32{10, 20, 30}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Generic", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Generic")

	var got []int32
	for batch, err := range ch.ReadBatch(BatchSize(10)) {
		requireNoError(t, err)
		// ReadBatch returns any which should be []int32
		slice, ok := batch.([]int32)
		if !ok {
			t.Fatalf("expected []int32 from ReadBatch, got %T", batch)
		}
		got = append(got, slice...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadBatch_Float64(t *testing.T) {
	expected := []float64{1.0, 2.0}
	fileBytes := buildStandardFile([]testChannelDef{
		float64Channel("GF64", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "GF64")

	var got []float64
	for batch, err := range ch.ReadBatch() {
		requireNoError(t, err)
		slice, ok := batch.([]float64)
		if !ok {
			t.Fatalf("expected []float64, got %T", batch)
		}
		got = append(got, slice...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadBatch_String(t *testing.T) {
	expected := []string{"alpha", "beta"}
	fileBytes := buildStandardFile([]testChannelDef{
		stringChannel("GStr", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "GStr")

	var got []string
	for batch, err := range ch.ReadBatch() {
		requireNoError(t, err)
		slice, ok := batch.([]string)
		if !ok {
			t.Fatalf("expected []string, got %T", batch)
		}
		got = append(got, slice...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadBatch_Bool(t *testing.T) {
	expected := []bool{true, false}
	fileBytes := buildStandardFile([]testChannelDef{
		boolChannel("GBool", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "GBool")

	var got []bool
	for batch, err := range ch.ReadBatch() {
		requireNoError(t, err)
		slice, ok := batch.([]bool)
		if !ok {
			t.Fatalf("expected []bool, got %T", batch)
		}
		got = append(got, slice...)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadMultipleChannelsDifferentTypes(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Ints", []int16{1, 2, 3}),
		float64Channel("Doubles", []float64{1.1, 2.2, 3.3}),
		boolChannel("Bools", []bool{true, false, true}),
		stringChannel("Strings", []string{"a", "bb", "ccc"}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	// Read int16 channel
	intCh := requireChannel(t, file, defaultGroupName, "Ints")
	intData, err := intCh.ReadInt16All()
	requireNoError(t, err)
	if !cmp.Equal([]int16{1, 2, 3}, intData) {
		t.Errorf("int data mismatch:\n%s", cmp.Diff([]int16{1, 2, 3}, intData))
	}

	// Read float64 channel
	floatCh := requireChannel(t, file, defaultGroupName, "Doubles")
	floatData, err := floatCh.ReadFloat64All()
	requireNoError(t, err)
	if !cmp.Equal([]float64{1.1, 2.2, 3.3}, floatData) {
		t.Errorf("float data mismatch:\n%s", cmp.Diff([]float64{1.1, 2.2, 3.3}, floatData))
	}

	// Read bool channel
	boolCh := requireChannel(t, file, defaultGroupName, "Bools")
	boolData, err := boolCh.ReadBoolAll()
	requireNoError(t, err)
	if !cmp.Equal([]bool{true, false, true}, boolData) {
		t.Errorf("bool data mismatch:\n%s", cmp.Diff([]bool{true, false, true}, boolData))
	}

	// Read string channel
	strCh := requireChannel(t, file, defaultGroupName, "Strings")
	strData, err := strCh.ReadStringAll()
	requireNoError(t, err)
	if !cmp.Equal([]string{"a", "bb", "ccc"}, strData) {
		t.Errorf("string data mismatch:\n%s", cmp.Diff([]string{"a", "bb", "ccc"}, strData))
	}
}

func TestChannel_ReadAllMultipleSegments(t *testing.T) {
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("Accum", DataTypeInt32, 3, nil),
	)
	raw1 := int32Bytes(10, 20, 30)
	raw2 := int32Bytes(40, 50, 60)
	raw3 := int32Bytes(70, 80, 90)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, raw1)
	tf.addSegment(standardSegmentTOC(), meta, raw2)
	tf.addSegment(standardSegmentTOC(), meta, raw3)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Accum")

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	expected := []int32{10, 20, 30, 40, 50, 60, 70, 80, 90}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadAllMultipleChunks(t *testing.T) {
	// 2 values per chunk, 8 values = 4 chunks in one segment
	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("ChunkedRead", DataTypeInt16, 2, nil),
	)
	rawData := int16Bytes(1, 2, 3, 4, 5, 6, 7, 8)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)

	file, err := readTDMSFromBytes(tf.build())
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "ChunkedRead")

	data, err := ch.ReadInt16All()
	requireNoError(t, err)

	expected := []int16{1, 2, 3, 4, 5, 6, 7, 8}
	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_Properties(t *testing.T) {
	// Note: writeProperty maps uint32 → type code 0x03 (DataTypeInt32),
	// so the value is readable via AsInt32(). We use uint32 intentionally.
	fileBytes := buildStandardFile([]testChannelDef{
		int16ChannelWithProps("Sensor", []int16{42}, map[string]any{
			"gain": uint32(100),
		}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Sensor")

	if len(ch.Properties) == 0 {
		t.Fatal("expected channel to have properties")
	}

	prop, ok := ch.Properties["gain"]
	if !ok {
		t.Fatal("property 'gain' not found")
	}

	val, err := prop.AsInt32()
	requireNoError(t, err)
	if val != 100 {
		t.Errorf("expected gain=100, got %d", val)
	}
}

func TestChannel_NameAndGroupName(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Pressure", []int16{1}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Pressure")

	if ch.Name != "Pressure" {
		t.Errorf("expected Name='Pressure', got %q", ch.Name)
	}
	if ch.GroupName != defaultGroupName {
		t.Errorf("expected GroupName=%q, got %q", defaultGroupName, ch.GroupName)
	}
}

func TestChannel_RawDataType(t *testing.T) {
	tests := []struct {
		name     string
		channel  testChannelDef
		expected DataType
	}{
		{"Int16", int16Channel("Int16", []int16{1}), DataTypeInt16},
		{"Int32", int32Channel("Int32", []int32{1}), DataTypeInt32},
		{"Uint8", uint8Channel("Uint8", []uint8{1}), DataTypeUint8},
		{"Uint32", uint32Channel("Uint32", []uint32{1}), DataTypeUint32},
		{"Float32", float32Channel("Float32", []float32{1.0}), DataTypeFloat32},
		{"Float64", float64Channel("Float64", []float64{1.0}), DataTypeFloat64},
		{"Bool", boolChannel("Bool", []bool{true}), DataTypeBool},
		{"String", stringChannel("String", []string{"x"}), DataTypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileBytes := buildStandardFile([]testChannelDef{tt.channel})

			file, err := readTDMSFromBytes(fileBytes)
			requireNoError(t, err)

			ch := requireChannel(t, file, defaultGroupName, tt.name)
			if ch.RawDataType != tt.expected {
				t.Errorf("expected RawDataType=%s, got %s", tt.expected, ch.RawDataType)
			}
		})
	}
}

func TestChannel_ReadWithCustomBatchSize(t *testing.T) {
	expected := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Batched", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Batched")

	// Read using streaming with small batch size — the individual value
	// iterator should still produce all values.
	var got []int32
	for val, err := range ch.ReadInt32(BatchSize(3)) {
		requireNoError(t, err)
		got = append(got, val)
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, got))
	}
}

func TestChannel_ReadConsistencyBetweenReadAllAndStream(t *testing.T) {
	// Verify ReadAll and stream iterator produce identical results
	expected := []int16{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5}
	fileBytes := buildStandardFile([]testChannelDef{
		int16Channel("Consistent", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Consistent")

	allData, err := ch.ReadInt16All()
	requireNoError(t, err)

	var streamData []int16
	for val, err := range ch.ReadInt16() {
		requireNoError(t, err)
		streamData = append(streamData, val)
	}

	if !cmp.Equal(allData, streamData) {
		t.Errorf("ReadAll vs Read stream mismatch:\n%s", cmp.Diff(allData, streamData))
	}
}

func TestChannel_ReadConsistencyBetweenReadAllAndBatch(t *testing.T) {
	expected := []int32{10, 20, 30, 40, 50}
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("AllVsBatch", expected),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "AllVsBatch")

	allData, err := ch.ReadInt32All()
	requireNoError(t, err)

	var batchData []int32
	for batch, err := range ch.ReadInt32Batch(BatchSize(2)) {
		requireNoError(t, err)
		batchData = append(batchData, batch...)
	}

	if !cmp.Equal(allData, batchData) {
		t.Errorf("ReadAll vs ReadBatch mismatch:\n%s", cmp.Diff(allData, batchData))
	}
}

func TestChannel_ReadLargeDataSet(t *testing.T) {
	n := 10_000
	values := make([]int32, n)
	for i := range values {
		values[i] = int32(i * 3)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Big", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Big")

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	if len(data) != n {
		t.Fatalf("expected %d values, got %d", n, len(data))
	}

	// Spot check
	if data[0] != 0 {
		t.Errorf("data[0]: expected 0, got %d", data[0])
	}
	if data[n-1] != int32((n-1)*3) {
		t.Errorf("data[%d]: expected %d, got %d", n-1, (n-1)*3, data[n-1])
	}
	if data[5000] != 15000 {
		t.Errorf("data[5000]: expected 15000, got %d", data[5000])
	}
}

func TestChannel_ReadSingleValue(t *testing.T) {
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("Solo", []int32{999}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "Solo")

	data, err := ch.ReadInt32All()
	requireNoError(t, err)

	if !cmp.Equal([]int32{999}, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff([]int32{999}, data))
	}
}

func TestChannel_ReadBatch_EarlyBreak(t *testing.T) {
	values := make([]int32, 100)
	for i := range values {
		values[i] = int32(i)
	}

	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("BreakBatch", values),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "BreakBatch")

	count := 0
	for _, err := range ch.ReadInt32Batch(BatchSize(10)) {
		requireNoError(t, err)
		count++
		if count == 2 {
			break
		}
	}

	if count != 2 {
		t.Errorf("expected to consume 2 batches, got %d", count)
	}
}

func TestChannel_ReadAcrossTwoChannelsInSameFile(t *testing.T) {
	// Ensure that reading one channel doesn't corrupt reading from another
	fileBytes := buildStandardFile([]testChannelDef{
		int32Channel("A", []int32{1, 2, 3, 4, 5}),
		int32Channel("B", []int32{10, 20, 30, 40, 50}),
	})

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	chA := requireChannel(t, file, defaultGroupName, "A")
	chB := requireChannel(t, file, defaultGroupName, "B")

	// Read A first, then B
	dataA, err := chA.ReadInt32All()
	requireNoError(t, err)

	dataB, err := chB.ReadInt32All()
	requireNoError(t, err)

	expectedA := []int32{1, 2, 3, 4, 5}
	expectedB := []int32{10, 20, 30, 40, 50}

	if !cmp.Equal(expectedA, dataA) {
		t.Errorf("channel A mismatch:\n%s", cmp.Diff(expectedA, dataA))
	}
	if !cmp.Equal(expectedB, dataB) {
		t.Errorf("channel B mismatch:\n%s", cmp.Diff(expectedB, dataB))
	}

	// Read them again to verify repeatability
	dataA2, err := chA.ReadInt32All()
	requireNoError(t, err)
	if !cmp.Equal(expectedA, dataA2) {
		t.Errorf("second read of A mismatch:\n%s", cmp.Diff(expectedA, dataA2))
	}
}

func TestChannel_ReadUint16All(t *testing.T) {
	var rawData bytes.Buffer
	expected := []uint16{0, 1, 65534, 65535}
	for _, v := range expected {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("U16", DataTypeUint16, uint64(len(expected)), nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData.Bytes())
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U16")

	data, err := ch.ReadUint16All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadInt64All(t *testing.T) {
	var rawData bytes.Buffer
	expected := []int64{-9223372036854775808, 0, 9223372036854775807}
	for _, v := range expected {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("I64", DataTypeInt64, uint64(len(expected)), nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData.Bytes())
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I64")

	data, err := ch.ReadInt64All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadUint64All(t *testing.T) {
	var rawData bytes.Buffer
	expected := []uint64{0, 1, 0xFFFFFFFFFFFFFFFF}
	for _, v := range expected {
		_ = binary.Write(&rawData, binary.LittleEndian, v)
	}

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("U64", DataTypeUint64, uint64(len(expected)), nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData.Bytes())
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "U64")

	data, err := ch.ReadUint64All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}

func TestChannel_ReadInt8All(t *testing.T) {
	rawData := []byte{0x01, 0x7F, 0x80, 0xFF} // 1, 127, -128, -1
	expected := []int8{1, 127, -128, -1}

	meta := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		channelMetadata("I8", DataTypeInt8, uint64(len(expected)), nil),
	)

	tf := newTestFile()
	tf.addSegment(standardSegmentTOC(), meta, rawData)
	fileBytes := tf.build()

	file, err := readTDMSFromBytes(fileBytes)
	requireNoError(t, err)

	ch := requireChannel(t, file, defaultGroupName, "I8")

	data, err := ch.ReadInt8All()
	requireNoError(t, err)

	if !cmp.Equal(expected, data) {
		t.Errorf("data mismatch:\n%s", cmp.Diff(expected, data))
	}
}
