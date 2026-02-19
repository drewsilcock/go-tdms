package tdms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/drewsilcock/go-tdms"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const testDataDir = "testdata"

// =============================================================================
// MANIFEST STRUCTURES
// =============================================================================

// Manifest represents the complete test manifest
type Manifest struct {
	Version     string     `json:"version"`
	Generated   string     `json:"generated"`
	Description string     `json:"description"`
	Tests       []TestCase `json:"tests"`
}

// TestCase represents a single test file and its expected values
type TestCase struct {
	ID          int                     `json:"id"`
	Name        string                  `json:"name"`
	Filename    string                  `json:"filename"`
	Description string                  `json:"description"`
	Features    []string                `json:"features"`
	Root        RootInfo                `json:"root"`
	Groups      []GroupInfo             `json:"groups"`
	Channels    []ChannelInfo           `json:"channels"`
	Scaling     map[string]ScalingInfo  `json:"scaling,omitempty"`
	Waveform    map[string]WaveformInfo `json:"waveform,omitempty"`
	Segments    []SegmentInfo           `json:"segments,omitempty"`
}

// RootInfo contains root-level properties
type RootInfo struct {
	Properties map[string]any `json:"properties"`
}

// GroupInfo represents expected group data
type GroupInfo struct {
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties"`
}

// ChannelInfo represents expected channel data
//
// RawDataType is the data type as stored in the TDMS file (matches ch.DataType).
// DataType is the output type after scaling is applied (determines which ReadXXX method to use).
// For channels without scaling, both fields will be the same.
type ChannelInfo struct {
	Group          string         `json:"group"`
	Channel        string         `json:"channel"`
	RawDataType    string         `json:"rawDataType"`    // Raw data type stored in TDMS file
	ScaledDataType string         `json:"scaledDataType"` // Output data type (after scaling)
	Length         int            `json:"length"`
	Data           any            `json:"data"` // Can be []any, nil, or other types
	Properties     map[string]any `json:"properties"`
	Statistics     *Statistics    `json:"statistics,omitempty"`
}

// Statistics for large data files where full data isn't included
type Statistics struct {
	Min     float64   `json:"min"`
	Max     float64   `json:"max"`
	Mean    float64   `json:"mean"`
	Std     float64   `json:"std"`
	First10 []float64 `json:"first10"`
	Last10  []float64 `json:"last10"`
}

// ScalingInfo represents scaling configuration and expected results
type ScalingInfo struct {
	Type           string    `json:"type"`
	Slope          float64   `json:"slope,omitempty"`
	Intercept      float64   `json:"intercept,omitempty"`
	Coefficients   []float64 `json:"coefficients,omitempty"`
	ExpectedScaled []float64 `json:"expectedScaled,omitempty"`
	Tolerance      float64   `json:"tolerance,omitempty"`
}

// WaveformInfo represents waveform properties
type WaveformInfo struct {
	StartOffset       float64   `json:"startOffset"`
	Increment         float64   `json:"increment"`
	Samples           int       `json:"samples"`
	ExpectedTimeRange []float64 `json:"expectedTimeRange,omitempty"`
}

// SegmentInfo represents segment-specific data
type SegmentInfo struct {
	Data []any `json:"data"`
}

// ComplexValue represents a complex number in JSON
type ComplexValue struct {
	Real float64 `json:"real"`
	Imag float64 `json:"imag"`
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func loadManifest(t *testing.T, testDataDir string) *Manifest {
	t.Helper()

	manifestPath := filepath.Join(testDataDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber() // Preserve precision for large integers
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	return &manifest
}

func hasFeature(tc TestCase, feature string) bool {
	return slices.Contains(tc.Features, feature)
}

type SignedInteger interface{ int8 | int16 | int32 | int64 }
type UnsignedInteger interface {
	uint8 | uint16 | uint32 | uint64
}
type Float interface{ float32 | float64 }
type Complex interface{ complex64 | complex128 }

// toFloat converts an any to a float
func toFloat[T Float](t *testing.T, value any) T {
	t.Helper()

	switch v := value.(type) {
	case json.Number:
		num, err := v.Float64()
		if err != nil {
			t.Fatalf("Failed to parse json.Number to float64: %v", err)
		}
		return T(num)
	case float64:
		return T(v)
	case int:
		return T(v)
	case string:
		// Handle special float values
		switch v {
		case "NaN":
			return T(math.NaN())
		case "Inf":
			return T(math.Inf(1))
		case "-Inf":
			return T(math.Inf(-1))
		default:
			t.Fatalf("Unknown string value: %s", v)
		}
	default:
		t.Fatalf("Unexpected type in float slice: %T", v)
	}

	return 0
}

func toFloat128(t *testing.T, value any) tdms.Float128 {
	t.Helper()

	switch v := value.(type) {
	case json.Number:
		num, err := v.Float64()
		if err != nil {
			t.Fatalf("Failed to parse json.Number to float64: %v", err)
		}
		return tdms.NewFloat128(num)
	case float64:
		return tdms.NewFloat128(v)
	case int:
		return tdms.NewFloat128(float64(v))
	case string:
		// Handle special float values
		switch v {
		case "NaN":
			return tdms.NewFloat128(math.NaN())
		case "Inf":
			return tdms.NewFloat128(math.Inf(1))
		case "-Inf":
			return tdms.NewFloat128(math.Inf(-1))
		default:
			t.Fatalf("Unknown string value: %s", v)
		}
	default:
		t.Fatalf("Unexpected type in float128: %T", v)
	}

	return tdms.NewFloat128(0)
}

// toSignedInteger converts an any slice to integer slice of specified type.
func toSignedInteger[T SignedInteger](t *testing.T, value any) T {
	t.Helper()

	switch val := value.(type) {
	case json.Number:
		num, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			t.Fatalf("Failed to parse json.Number to integer: %v", err)
		}
		return T(num)
	case float64:
		return T(val)
	case int:
		return T(val)
	default:
		t.Fatalf("Expected numeric value, got %T", val)
	}

	return 0
}

func toUnsignedInteger[T UnsignedInteger](t *testing.T, value any) T {
	t.Helper()

	switch val := value.(type) {
	case json.Number:
		num, err := strconv.ParseUint(string(val), 10, 64)
		if err != nil {
			t.Fatalf("Failed to parse json.Number to unsigned integer: %v", err)
		}
		return T(num)
	case float64:
		return T(val)
	case int:
		return T(val)
	default:
		t.Fatalf("Expected numeric value, got %T", val)

	}

	return 0
}

// toComplex converts an any containing real & imag values in map to complex64 or complex128.
func toComplex[T Complex](t *testing.T, value any) T {
	t.Helper()

	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("Expected map for complex value, got %T", value)
	}

	var realVal, imagVal float64
	if realNum, ok := m["real"].(json.Number); ok {
		var err error
		realVal, err = realNum.Float64()
		if err != nil {
			t.Fatalf("Failed to parse real part as float64: %v", err)
		}
	} else if realFloat, ok := m["real"].(float64); ok {
		realVal = realFloat
	}

	if imagNum, ok := m["imag"].(json.Number); ok {
		var err error
		imagVal, err = imagNum.Float64()
		if err != nil {
			t.Fatalf("Failed to parse imag part as float64: %v", err)
		}
	} else if imagFloat, ok := m["imag"].(float64); ok {
		imagVal = imagFloat
	}

	return T(complex(realVal, imagVal))
}

func toTimestamp(t *testing.T, value any) tdms.Timestamp {
	t.Helper()

	timestampIso, ok := value.(string)
	if !ok {
		t.Fatalf("Expected string for timestamp value, got %T", value)
	}

	ts, err := time.Parse(time.RFC3339Nano, timestampIso)
	if err == nil {
		return tdms.NewTimestamp(ts)
	}

	ts, err = time.Parse(time.RFC3339, timestampIso)
	if err == nil {
		return tdms.NewTimestamp(ts)
	}

	ts, err = time.Parse("2006-01-02T15:04:05", timestampIso)
	if err == nil {
		return tdms.NewTimestamp(ts)
	}

	t.Fatalf("Failed to parse timestamp as time.Time: %v", err)
	return tdms.Timestamp{}
}

// =============================================================================
// MAIN PARAMETERIZED TEST
// =============================================================================

func TestTDMSFilesFromManifest(t *testing.T) {
	// Check if test data directory exists
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skipf("Test data directory %s does not exist. Run the Python generator first.", testDataDir)
	}

	manifest := loadManifest(t, testDataDir)

	for _, tc := range manifest.Tests {
		tc := tc // capture range variable
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(testDataDir, tc.Filename)

			// Skip if file doesn't exist
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Skipf("Test file %s does not exist", tc.Filename)
			}

			// Open the TDMS file
			f, err := tdms.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open TDMS file %s: %v", tc.Filename, err)
			}
			defer func(f *tdms.File) {
				_ = f.Close()
			}(f)

			// Run sub-tests
			t.Run("Groups", func(t *testing.T) {
				testGroups(t, f, tc)
			})

			t.Run("Channels", func(t *testing.T) {
				testChannels(t, f, tc)
			})

			t.Run("RootProperties", func(t *testing.T) {
				testRootProperties(t, f, tc)
			})

			t.Run("ChannelData", func(t *testing.T) {
				testChannelData(t, f, tc)
			})

			t.Run("ChannelProperties", func(t *testing.T) {
				testChannelProperties(t, f, tc)
			})
		})
	}
}

// =============================================================================
// SUB-TESTS
// =============================================================================

func testGroups(t *testing.T, f *tdms.File, tc TestCase) {
	// Check number of groups
	if len(f.Groups) != len(tc.Groups) {
		t.Errorf("Expected %d groups, got %d", len(tc.Groups), len(f.Groups))
		return
	}

	// Check each expected group exists
	for _, expectedGroup := range tc.Groups {
		group, exists := f.Groups[expectedGroup.Name]
		if !exists {
			t.Errorf("Expected group %q not found", expectedGroup.Name)
			continue
		}

		// Check group name
		if group.Name != expectedGroup.Name {
			t.Errorf("Group name mismatch: expected %q, got %q", expectedGroup.Name, group.Name)
		}

		// Check group properties
		for propName, expectedValue := range expectedGroup.Properties {
			prop, exists := group.Properties[propName]
			if !exists {
				t.Errorf("Group %q: expected property %q not found", expectedGroup.Name, propName)
				continue
			}
			if !comparePropertyValue(prop.Value, expectedValue) {
				t.Errorf("Group %q property %q: expected %v, got %v",
					expectedGroup.Name, propName, expectedValue, prop.Value)
			}
		}
	}
}

func testChannels(t *testing.T, f *tdms.File, tc TestCase) {
	// Build a map of expected channels by group
	expectedByGroup := make(map[string][]ChannelInfo)
	for _, ch := range tc.Channels {
		expectedByGroup[ch.Group] = append(expectedByGroup[ch.Group], ch)
	}

	// Check each group has the expected channels
	for groupName, expectedChannels := range expectedByGroup {
		group, exists := f.Groups[groupName]
		if !exists {
			t.Errorf("Group %q not found", groupName)
			continue
		}

		if len(group.Channels) != len(expectedChannels) {
			t.Errorf("Group %q: expected %d channels, got %d",
				groupName, len(expectedChannels), len(group.Channels))
		}

		for _, expectedCh := range expectedChannels {
			ch, exists := group.Channels[expectedCh.Channel]
			if !exists {
				t.Errorf("Channel %q/%q not found", groupName, expectedCh.Channel)
				continue
			}

			if ch.Name != expectedCh.Channel {
				t.Errorf("Channel name mismatch: expected %q, got %q", expectedCh.Channel, ch.Name)
			}

			if ch.GroupName != groupName {
				t.Errorf("Channel %q: group name mismatch: expected %q, got %q",
					expectedCh.Channel, groupName, ch.GroupName)
			}

			wantRawDataType := parseDataType(expectedCh.RawDataType)
			if ch.RawDataType != wantRawDataType {
				t.Errorf("Channel %q/%q: raw data type mismatch: expected %v, got %v",
					groupName, expectedCh.Channel, wantRawDataType, ch.RawDataType)
			}

			wantScaledDataType := parseDataType(expectedCh.ScaledDataType)
			if ch.ScaledDataType != wantScaledDataType {
				t.Errorf("Channel %q/%q: scaled data type mismatch: expected %v, got %v",
					groupName, expectedCh.Channel, wantScaledDataType, ch.ScaledDataType)
			}

			// Check raw data type (stored in file)
			// Use RawDataType if present, otherwise fall back to DataType for backwards compatibility
			rawDataTypeStr := expectedCh.RawDataType
			if rawDataTypeStr == "" {
				rawDataTypeStr = expectedCh.ScaledDataType
			}
			expectedRawDataType := parseDataType(rawDataTypeStr)
			if expectedRawDataType != tdms.DataTypeVoid && ch.RawDataType != expectedRawDataType {
				t.Errorf("Channel %q/%q: raw data type mismatch: expected %v, got %v",
					groupName, expectedCh.Channel, expectedRawDataType, ch.RawDataType)
			}
		}
	}
}

func testRootProperties(t *testing.T, f *tdms.File, tc TestCase) {
	for propName, expectedValue := range tc.Root.Properties {
		prop, exists := f.Properties[propName]
		if !exists {
			t.Errorf("Expected root property %q not found", propName)
			continue
		}
		if !comparePropertyValue(prop.Value, expectedValue) {
			t.Errorf("Root property %q: expected %v (%T), got %v (%T)",
				propName, expectedValue, expectedValue, prop.Value, prop.Value)
		}
	}
}

func testChannelData(t *testing.T, f *tdms.File, tc TestCase) {
	for _, expectedCh := range tc.Channels {
		group, exists := f.Groups[expectedCh.Group]
		if !exists {
			continue
		}

		ch, exists := group.Channels[expectedCh.Channel]
		if !exists {
			continue
		}

		wantRawDataType := parseDataType(expectedCh.RawDataType)
		if ch.RawDataType != wantRawDataType {
			t.Errorf(
				"Channel %s/%s: expected raw data type %s, got %s",
				expectedCh.Group,
				expectedCh.Channel,
				wantRawDataType,
				ch.RawDataType,
			)
		}

		wantScaledDataType := parseDataType(expectedCh.ScaledDataType)
		if ch.ScaledDataType != wantScaledDataType {
			t.Errorf(
				"Channel %s/%s: expected data type %s, got %s",
				expectedCh.Group,
				expectedCh.Channel,
				wantScaledDataType,
				ch.ScaledDataType,
			)
		}

		// Skip if no data to compare (e.g., large data files)
		if expectedCh.Data == nil {
			// For large data, check statistics if available
			if expectedCh.Statistics != nil {
				testChannelStatistics(t, &ch, expectedCh)
			}
			continue
		}

		// Scaling is tested separately, so we disable it here.
		gotData, err := ch.ReadAll(tdms.WithScaling(false))
		if err != nil {
			t.Errorf("Failed to read channel data: %v", err)
			continue
		}

		opts := cmp.Options{cmpopts.EquateNaNs()}
		switch ch.ScaledDataType {
		case tdms.DataTypeFloat32, tdms.DataTypeFloat32WithUnit, tdms.DataTypeComplex64:
			opts = append(opts, cmpopts.EquateApprox(1e-6, 1e-6))
		case tdms.DataTypeFloat64, tdms.DataTypeFloat64WithUnit, tdms.DataTypeComplex128:
			opts = append(opts, cmpopts.EquateApprox(1e-9, 1e-9))
		}

		wantData := convertSlice(t, expectedCh.Data, wantRawDataType)
		if !cmp.Equal(wantData, gotData, opts) {
			t.Errorf(
				"Channel %s/%s: data mismatch: %s",
				expectedCh.Group,
				expectedCh.Channel,
				cmp.Diff(wantData, gotData, opts),
			)
		}
	}
}

func convertSlice(t *testing.T, data any, dataType tdms.DataType) any {
	t.Helper()

	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		t.Fatalf("expected data to be slice, got %T", data)
	}

	var out any

	switch dataType {
	case tdms.DataTypeInt8:
		slice := make([]int8, dataValue.Len())
		for i := range slice {
			slice[i] = toSignedInteger[int8](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeInt16:
		slice := make([]int16, dataValue.Len())
		for i := range slice {
			slice[i] = toSignedInteger[int16](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeInt32:
		slice := make([]int32, dataValue.Len())
		for i := range slice {
			slice[i] = toSignedInteger[int32](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeInt64:
		slice := make([]int64, dataValue.Len())
		for i := range slice {
			slice[i] = toSignedInteger[int64](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeUint8:
		slice := make([]uint8, dataValue.Len())
		for i := range slice {
			slice[i] = toUnsignedInteger[uint8](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeUint16:
		slice := make([]uint16, dataValue.Len())
		for i := range slice {
			slice[i] = toUnsignedInteger[uint16](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeUint32:
		slice := make([]uint32, dataValue.Len())
		for i := range slice {
			slice[i] = toUnsignedInteger[uint32](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeUint64:
		slice := make([]uint64, dataValue.Len())
		for i := range slice {
			slice[i] = toUnsignedInteger[uint64](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeFloat32:
		slice := make([]float32, dataValue.Len())
		for i := range slice {
			slice[i] = toFloat[float32](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeFloat64:
		slice := make([]float64, dataValue.Len())
		for i := range slice {
			slice[i] = toFloat[float64](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeFloat128:
		slice := make([]tdms.Float128, dataValue.Len())
		for i := range slice {
			slice[i] = toFloat128(t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeBool:
		slice := make([]bool, dataValue.Len())
		for i := range slice {
			slice[i] = dataValue.Index(i).Interface().(bool)
		}
		out = slice
	case tdms.DataTypeString:
		slice := make([]string, dataValue.Len())
		for i := range slice {
			slice[i] = dataValue.Index(i).Interface().(string)
		}
		out = slice
	case tdms.DataTypeTimestamp:
		slice := make([]tdms.Timestamp, dataValue.Len())
		for i := range slice {
			slice[i] = toTimestamp(t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeComplex64:
		slice := make([]complex64, dataValue.Len())
		for i := range slice {
			slice[i] = toComplex[complex64](t, dataValue.Index(i).Interface())
		}
		out = slice
	case tdms.DataTypeComplex128:
		slice := make([]complex128, dataValue.Len())
		for i := range slice {
			slice[i] = toComplex[complex128](t, dataValue.Index(i).Interface())
		}
		out = slice
	default:
		t.Fatalf("unsupported data type: %v", dataType)
	}

	return out
}

func testChannelProperties(t *testing.T, f *tdms.File, tc TestCase) {
	for _, expectedCh := range tc.Channels {
		group, exists := f.Groups[expectedCh.Group]
		if !exists {
			continue
		}

		ch, exists := group.Channels[expectedCh.Channel]
		if !exists {
			continue
		}

		for propName, expectedValue := range expectedCh.Properties {
			prop, exists := ch.Properties[propName]
			if !exists {
				t.Errorf("Channel %s/%s: expected property %q not found",
					expectedCh.Group, expectedCh.Channel, propName)
				continue
			}
			if !comparePropertyValue(prop.Value, expectedValue) {
				t.Errorf("Channel %s/%s property %q: expected %v, got %v",
					expectedCh.Group, expectedCh.Channel, propName, expectedValue, prop.Value)
			}
		}
	}
}

// =============================================================================
// DATA TYPE SPECIFIC TESTS
// =============================================================================

func testChannelStatistics(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	want := expected.Statistics
	if expected.Statistics == nil {
		return
	}

	data, err := ch.ReadFloat64All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read data for statistics: %v",
			expected.Group, expected.Channel, err)
		return
	}

	if len(data) != expected.Length {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, expected.Length, len(data))
	}

	// Compute and check statistics
	got := &Statistics{}
	sum := 0.0
	got.Min = math.Inf(1)
	got.Max = math.Inf(-1)
	for _, v := range data {
		sum += v
		if v < got.Min {
			got.Min = v
		}
		if v > got.Max {
			got.Max = v
		}
	}
	got.Mean = sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - got.Mean
		variance += diff * diff
	}
	got.Std = math.Sqrt(variance / float64(len(data)))

	got.First10 = make([]float64, 10)
	copy(got.First10, data[:10])

	got.Last10 = make([]float64, 10)
	copy(got.Last10, data[len(data)-10:])

	if !cmp.Equal(want, got, cmpopts.EquateApprox(1e-9, 1e-9)) {
		t.Errorf(
			"Channel %s/%s: statistics mismatch: %v",
			expected.Group, expected.Channel,
			cmp.Diff(want, got, cmpopts.EquateApprox(1e-9, 1e-9)),
		)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// parseDataType converts string TDMS data type name to corresponding tdms.DataType
func parseDataType(s string) tdms.DataType {
	switch s {
	case "int8":
		return tdms.DataTypeInt8
	case "int16":
		return tdms.DataTypeInt16
	case "int32":
		return tdms.DataTypeInt32
	case "int64":
		return tdms.DataTypeInt64
	case "uint8":
		return tdms.DataTypeUint8
	case "uint16":
		return tdms.DataTypeUint16
	case "uint32":
		return tdms.DataTypeUint32
	case "uint64":
		return tdms.DataTypeUint64
	case "float32":
		return tdms.DataTypeFloat32
	case "float64":
		return tdms.DataTypeFloat64
	case "string":
		return tdms.DataTypeString
	case "boolean":
		return tdms.DataTypeBool
	case "timestamp":
		return tdms.DataTypeTimestamp
	case "complex64":
		return tdms.DataTypeComplex64
	case "complex128":
		return tdms.DataTypeComplex128
	default:
		panic(fmt.Sprintf("unknown data type %q", s))
	}
}

// comparePropertyValue compares a property value from the file with expected value from JSON
func comparePropertyValue(got any, want any) bool {
	switch w := want.(type) {
	case json.Number:
		// Handle json.Number from UseNumber() decoder
		// Try to parse as float64 for comparison
		expectedFloat, err := w.Float64()
		if err != nil {
			return false
		}
		switch g := got.(type) {
		case float64:
			return cmp.Equal(g, expectedFloat, cmpopts.EquateApprox(1e-9, 1e-9))
		case float32:
			return cmp.Equal(float64(g), expectedFloat, cmpopts.EquateApprox(1e-6, 1e-6))
		case int:
			return float64(g) == expectedFloat
		case int32:
			return float64(g) == expectedFloat
		case int64:
			return float64(g) == expectedFloat
		case uint32:
			return float64(g) == expectedFloat
		case uint64:
			return float64(g) == expectedFloat
		}
	case float64:
		// JSON numbers might still be float64 in some cases
		switch a := got.(type) {
		case float64:
			return cmp.Equal(a, w, cmpopts.EquateApprox(1e-9, 1e-9))
		case float32:
			return cmp.Equal(float64(a), w, cmpopts.EquateApprox(1e-6, 1e-6))
		case int:
			return float64(a) == w
		case int32:
			return float64(a) == w
		case int64:
			return float64(a) == w
		case uint32:
			return float64(a) == w
		case uint64:
			return float64(a) == w
		}
	case string:
		if a, ok := got.(string); ok {
			return a == w
		}
	case bool:
		if a, ok := got.(bool); ok {
			return a == w
		}
	case nil:
		return got == nil
	}
	return false
}

// =============================================================================
// FEATURE-SPECIFIC TESTS
// =============================================================================

func TestMultipleSegments(t *testing.T) {
	manifest := loadManifest(t, testDataDir)

	for _, tc := range manifest.Tests {
		if !hasFeature(tc, "multiple_segments") {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			filePath := filepath.Join(testDataDir, tc.Filename)
			f, err := tdms.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer func(f *tdms.File) {
				_ = f.Close()
			}(f)

			// For multiple segment tests, verify that all data from all segments
			// is correctly concatenated
			for _, expectedCh := range tc.Channels {
				group, exists := f.Groups[expectedCh.Group]
				if !exists {
					t.Errorf("Group %s not found", expectedCh.Group)
					continue
				}

				ch, exists := group.Channels[expectedCh.Channel]
				if !exists {
					t.Errorf("Channel %s not found", expectedCh.Channel)
					continue
				}

				// Read all data and verify length matches expected
				switch expectedCh.ScaledDataType {
				case "int32":
					got, err := ch.ReadInt32All()
					if err != nil {
						t.Errorf("Failed to read data: %v", err)
						continue
					}

					want := convertSlice(t, expectedCh.Data, tdms.DataTypeInt32)
					if !cmp.Equal(want, got) {
						t.Errorf("Data mismatch: %v", cmp.Diff(want, got))
					}
				}
			}
		})
	}
}

func TestScalingProperties(t *testing.T) {
	manifest := loadManifest(t, testDataDir)

	for _, tc := range manifest.Tests {
		if !hasFeature(tc, "scaling") {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			filePath := filepath.Join(testDataDir, tc.Filename)
			f, err := tdms.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer func(f *tdms.File) {
				_ = f.Close()
			}(f)

			// Verify scaling properties are correctly parsed
			for _, expectedCh := range tc.Channels {
				group, exists := f.Groups[expectedCh.Group]
				if !exists {
					continue
				}

				ch, exists := group.Channels[expectedCh.Channel]
				if !exists {
					continue
				}

				// Check for NI scaling properties
				if numScales, exists := expectedCh.Properties["NI_Number_Of_Scales"]; exists {
					prop, found := ch.Properties["NI_Number_Of_Scales"]
					if !found {
						t.Errorf("Channel %s/%s: NI_Number_Of_Scales property not found",
							expectedCh.Group, expectedCh.Channel)
						continue
					}

					if !comparePropertyValue(prop.Value, numScales) {
						t.Errorf("Channel %s/%s: NI_Number_Of_Scales mismatch: expected %v, got %v",
							expectedCh.Group, expectedCh.Channel, numScales, prop.Value)
					}
				}

				// Check scale type
				if scaleType, exists := expectedCh.Properties["NI_Scale[0]_Scale_Type"]; exists {
					prop, found := ch.Properties["NI_Scale[0]_Scale_Type"]
					if !found {
						t.Errorf("Channel %s/%s: NI_Scale[0]_Scale_Type property not found",
							expectedCh.Group, expectedCh.Channel)
						continue
					}

					if prop.Value != scaleType {
						t.Errorf("Channel %s/%s: NI_Scale[0]_Scale_Type mismatch: expected %v, got %v",
							expectedCh.Group, expectedCh.Channel, scaleType, prop.Value)
					}
				}
			}
		})
	}
}

func TestEmptyChannels(t *testing.T) {
	manifest := loadManifest(t, testDataDir)

	for _, tc := range manifest.Tests {
		if !hasFeature(tc, "empty_channel") {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			filePath := filepath.Join(testDataDir, tc.Filename)
			f, err := tdms.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer func(f *tdms.File) {
				_ = f.Close()
			}(f)

			for _, expectedCh := range tc.Channels {
				if expectedCh.Length != 0 {
					continue
				}

				group, exists := f.Groups[expectedCh.Group]
				if !exists {
					t.Errorf("Group %s not found", expectedCh.Group)
					continue
				}

				ch, exists := group.Channels[expectedCh.Channel]
				if !exists {
					t.Errorf("Channel %s not found", expectedCh.Channel)
					continue
				}

				// Verify empty channel returns empty data
				data, err := ch.ReadFloat64All()
				if err != nil {
					t.Errorf("Failed to read empty channel: %v", err)
					continue
				}

				if len(data) != 0 {
					t.Errorf("Expected empty channel, got %d values", len(data))
				}
			}
		})
	}
}

func TestSpecialCharacterNames(t *testing.T) {
	manifest := loadManifest(t, testDataDir)

	for _, tc := range manifest.Tests {
		if !hasFeature(tc, "unicode_names") && !hasFeature(tc, "special_characters") {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			filePath := filepath.Join(testDataDir, tc.Filename)
			f, err := tdms.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer func(f *tdms.File) {
				_ = f.Close()
			}(f)

			// Verify all groups with special names are found
			for _, expectedGroup := range tc.Groups {
				if _, exists := f.Groups[expectedGroup.Name]; !exists {
					t.Errorf("Group with special name %q not found", expectedGroup.Name)
				}
			}

			// Verify all channels with special names are found
			for _, expectedCh := range tc.Channels {
				group, exists := f.Groups[expectedCh.Group]
				if !exists {
					continue
				}

				if _, exists := group.Channels[expectedCh.Channel]; !exists {
					t.Errorf("Channel with special name %q not found in group %q",
						expectedCh.Channel, expectedCh.Group)
				}
			}
		})
	}
}
