package tdms

import (
	"encoding/json"
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/drewsilcock/go-tdms"
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
type ChannelInfo struct {
	Group      string         `json:"group"`
	Channel    string         `json:"channel"`
	DataType   string         `json:"dataType"`
	Length     int            `json:"length"`
	Data       any            `json:"data"` // Can be []any, nil, or other types
	Properties map[string]any `json:"properties"`
	Statistics *Statistics    `json:"statistics,omitempty"`
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
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	return &manifest
}

func hasFeature(tc TestCase, feature string) bool {
	return slices.Contains(tc.Features, feature)
}

// toFloat64Slice converts an any slice to float64 slice
func toFloat64Slice(t *testing.T, data any) []float64 {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]float64, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case float64:
			result[i] = val
		case int:
			result[i] = float64(val)
		case string:
			// Handle special float values
			switch val {
			case "NaN":
				result[i] = math.NaN()
			case "Inf":
				result[i] = math.Inf(1)
			case "-Inf":
				result[i] = math.Inf(-1)
			default:
				t.Fatalf("Unknown string value: %s", val)
			}
		default:
			t.Fatalf("Unexpected type in float slice: %T", v)
		}
	}
	return result
}

// toInt32Slice converts an any slice to int32 slice
func toInt32Slice(t *testing.T, data any) []int32 {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]int32, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case float64:
			result[i] = int32(val)
		case int:
			result[i] = int32(val)
		default:
			t.Fatalf("Unexpected type in int32 slice: %T", v)
		}
	}
	return result
}

// toInt64Slice converts an any slice to int64 slice
func toInt64Slice(t *testing.T, data any) []int64 {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]int64, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case float64:
			result[i] = int64(val)
		case int:
			result[i] = int64(val)
		default:
			t.Fatalf("Unexpected type in int64 slice: %T", v)
		}
	}
	return result
}

// toUint8Slice converts an any slice to uint8 slice
func toUint8Slice(t *testing.T, data any) []uint8 {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]uint8, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case float64:
			result[i] = uint8(val)
		case int:
			result[i] = uint8(val)
		default:
			t.Fatalf("Unexpected type in uint8 slice: %T", v)
		}
	}
	return result
}

// toUint32Slice converts an any slice to uint32 slice
func toUint32Slice(t *testing.T, data any) []uint32 {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]uint32, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case float64:
			result[i] = uint32(val)
		case int:
			result[i] = uint32(val)
		default:
			t.Fatalf("Unexpected type in uint32 slice: %T", v)
		}
	}
	return result
}

// toStringSlice converts an any slice to string slice
func toStringSlice(t *testing.T, data any) []string {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]string, len(slice))
	for i, v := range slice {
		str, ok := v.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", v)
		}
		result[i] = str
	}
	return result
}

// toBoolSlice converts an any slice to bool slice
func toBoolSlice(t *testing.T, data any) []bool {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]bool, len(slice))
	for i, v := range slice {
		b, ok := v.(bool)
		if !ok {
			t.Fatalf("Expected bool, got %T", v)
		}
		result[i] = b
	}
	return result
}

// toComplex128Slice converts an any slice of complex values to complex128 slice
func toComplex128Slice(t *testing.T, data any) []complex128 {
	t.Helper()

	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", data)
	}

	result := make([]complex128, len(slice))
	for i, v := range slice {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("Expected map for complex value, got %T", v)
		}
		real, _ := m["real"].(float64)
		imag, _ := m["imag"].(float64)
		result[i] = complex(real, imag)
	}
	return result
}

// floatEquals compares two floats, handling NaN and Inf
func floatEquals(a, b float64, tolerance float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return true
	}
	if math.IsInf(a, -1) && math.IsInf(b, -1) {
		return true
	}
	if tolerance == 0 {
		tolerance = 1e-9
	}
	return math.Abs(a-b) <= tolerance
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
			defer f.Close()

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

			// Check channel name
			if ch.Name != expectedCh.Channel {
				t.Errorf("Channel name mismatch: expected %q, got %q",
					expectedCh.Channel, ch.Name)
			}

			// Check group name
			if ch.GroupName != groupName {
				t.Errorf("Channel %q: group name mismatch: expected %q, got %q",
					expectedCh.Channel, groupName, ch.GroupName)
			}

			// Check data type
			expectedDataType := mapDataType(expectedCh.DataType)
			if expectedDataType != tdms.DataTypeVoid && ch.DataType != expectedDataType {
				t.Errorf("Channel %q/%q: data type mismatch: expected %v, got %v",
					groupName, expectedCh.Channel, expectedDataType, ch.DataType)
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

		// Skip if no data to compare (e.g., large data files)
		if expectedCh.Data == nil {
			// For large data, check statistics if available
			if expectedCh.Statistics != nil {
				testChannelStatistics(t, &ch, expectedCh)
			}
			continue
		}

		// Compare actual data based on type
		switch expectedCh.DataType {
		case "int8":
			testInt8Data(t, &ch, expectedCh)
		case "int16":
			testInt16Data(t, &ch, expectedCh)
		case "int32":
			testInt32Data(t, &ch, expectedCh)
		case "int64":
			testInt64Data(t, &ch, expectedCh)
		case "uint8":
			testUint8Data(t, &ch, expectedCh)
		case "uint16":
			testUint16Data(t, &ch, expectedCh)
		case "uint32":
			testUint32Data(t, &ch, expectedCh)
		case "uint64":
			testUint64Data(t, &ch, expectedCh)
		case "float32":
			testFloat32Data(t, &ch, expectedCh)
		case "float64":
			testFloat64Data(t, &ch, expectedCh)
		case "string":
			testStringData(t, &ch, expectedCh)
		case "boolean":
			testBoolData(t, &ch, expectedCh)
		case "complex64":
			testComplex64Data(t, &ch, expectedCh)
		case "complex128":
			testComplex128Data(t, &ch, expectedCh)
		case "timestamp":
			// Timestamp testing would require parsing ISO format
			t.Logf("Skipping timestamp data comparison for %s/%s", expectedCh.Group, expectedCh.Channel)
		default:
			t.Logf("Unknown data type %s for %s/%s", expectedCh.DataType, expectedCh.Group, expectedCh.Channel)
		}
	}
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

func testInt8Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataInt8All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read int8 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toInt32Slice(t, expected.Data) // JSON numbers are float64, convert via int32
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if int8(expectedData[i]) != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testInt16Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataInt16All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read int16 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toInt32Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if int16(expectedData[i]) != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testInt32Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataInt32All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read int32 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toInt32Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if expectedData[i] != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testInt64Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataInt64All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read int64 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toInt64Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if expectedData[i] != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testUint8Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataUint8All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read uint8 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toUint8Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if expectedData[i] != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testUint16Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataUint16All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read uint16 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toUint32Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if uint16(expectedData[i]) != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testUint32Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataUint32All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read uint32 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toUint32Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if expectedData[i] != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testUint64Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataUint64All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read uint64 data: %v", expected.Group, expected.Channel, err)
		return
	}

	// JSON numbers might lose precision for uint64, so we need careful handling
	slice, ok := expected.Data.([]any)
	if !ok {
		t.Errorf("Expected []any for data, got %T", expected.Data)
		return
	}

	if len(data) != len(slice) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(slice), len(data))
		return
	}

	for i := range data {
		var expectedVal uint64
		switch v := slice[i].(type) {
		case float64:
			expectedVal = uint64(v)
		case int:
			expectedVal = uint64(v)
		}
		if expectedVal != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %d, got %d",
				expected.Group, expected.Channel, i, expectedVal, data[i])
		}
	}
}

func testFloat32Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataFloat32All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read float32 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toFloat64Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if !floatEquals(float64(data[i]), expectedData[i], 1e-6) {
			t.Errorf("Channel %s/%s[%d]: expected %v, got %v",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testFloat64Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataFloat64All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read float64 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toFloat64Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if !floatEquals(data[i], expectedData[i], 1e-9) {
			t.Errorf("Channel %s/%s[%d]: expected %v, got %v",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testStringData(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataStringAll()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read string data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toStringSlice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if expectedData[i] != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %q, got %q",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testBoolData(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataBoolAll()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read bool data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toBoolSlice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if expectedData[i] != data[i] {
			t.Errorf("Channel %s/%s[%d]: expected %v, got %v",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testComplex64Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataComplex64All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read complex64 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toComplex128Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		expected64 := complex64(expectedData[i])
		if !cmplx.IsNaN(complex128(data[i])) && !cmplx.IsNaN(complex128(expected64)) {
			if real(data[i]) != real(expected64) || imag(data[i]) != imag(expected64) {
				t.Errorf("Channel %s/%s[%d]: expected %v, got %v",
					expected.Group, expected.Channel, i, expected64, data[i])
			}
		}
	}
}

func testComplex128Data(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	data, err := ch.ReadDataComplex128All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read complex128 data: %v", expected.Group, expected.Channel, err)
		return
	}

	expectedData := toComplex128Slice(t, expected.Data)
	if len(data) != len(expectedData) {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, len(expectedData), len(data))
		return
	}

	for i := range data {
		if !floatEquals(real(data[i]), real(expectedData[i]), 1e-9) ||
			!floatEquals(imag(data[i]), imag(expectedData[i]), 1e-9) {
			t.Errorf("Channel %s/%s[%d]: expected %v, got %v",
				expected.Group, expected.Channel, i, expectedData[i], data[i])
		}
	}
}

func testChannelStatistics(t *testing.T, ch *tdms.Channel, expected ChannelInfo) {
	if expected.Statistics == nil {
		return
	}

	data, err := ch.ReadDataFloat64All()
	if err != nil {
		t.Errorf("Channel %s/%s: failed to read data for statistics: %v",
			expected.Group, expected.Channel, err)
		return
	}

	if len(data) != expected.Length {
		t.Errorf("Channel %s/%s: length mismatch: expected %d, got %d",
			expected.Group, expected.Channel, expected.Length, len(data))
	}

	// Check first 10 values
	for i, expectedVal := range expected.Statistics.First10 {
		if i >= len(data) {
			break
		}
		if !floatEquals(data[i], expectedVal, 1e-9) {
			t.Errorf("Channel %s/%s first10[%d]: expected %v, got %v",
				expected.Group, expected.Channel, i, expectedVal, data[i])
		}
	}

	// Check last 10 values
	for i, expectedVal := range expected.Statistics.Last10 {
		idx := len(data) - 10 + i
		if idx < 0 || idx >= len(data) {
			continue
		}
		if !floatEquals(data[idx], expectedVal, 1e-9) {
			t.Errorf("Channel %s/%s last10[%d]: expected %v, got %v",
				expected.Group, expected.Channel, i, expectedVal, data[idx])
		}
	}

	// Compute and check statistics
	var sum, min, max float64
	min = math.Inf(1)
	max = math.Inf(-1)
	for _, v := range data {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	mean := sum / float64(len(data))

	if !floatEquals(min, expected.Statistics.Min, 1e-6) {
		t.Errorf("Channel %s/%s: min mismatch: expected %v, got %v",
			expected.Group, expected.Channel, expected.Statistics.Min, min)
	}
	if !floatEquals(max, expected.Statistics.Max, 1e-6) {
		t.Errorf("Channel %s/%s: max mismatch: expected %v, got %v",
			expected.Group, expected.Channel, expected.Statistics.Max, max)
	}
	if !floatEquals(mean, expected.Statistics.Mean, 1e-6) {
		t.Errorf("Channel %s/%s: mean mismatch: expected %v, got %v",
			expected.Group, expected.Channel, expected.Statistics.Mean, mean)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// mapDataType maps string data type names to tdms.DataType
func mapDataType(s string) tdms.DataType {
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
		return tdms.DataTypeTime
	case "complex64":
		return tdms.DataTypeComplex64
	case "complex128":
		return tdms.DataTypeComplex128
	default:
		return tdms.DataTypeVoid
	}
}

// comparePropertyValue compares a property value from the file with expected value from JSON
func comparePropertyValue(actual any, expected any) bool {
	switch e := expected.(type) {
	case float64:
		// JSON numbers are always float64
		switch a := actual.(type) {
		case float64:
			return floatEquals(a, e, 1e-9)
		case float32:
			return floatEquals(float64(a), e, 1e-6)
		case int:
			return float64(a) == e
		case int32:
			return float64(a) == e
		case int64:
			return float64(a) == e
		case uint32:
			return float64(a) == e
		case uint64:
			return float64(a) == e
		}
	case string:
		if a, ok := actual.(string); ok {
			return a == e
		}
	case bool:
		if a, ok := actual.(bool); ok {
			return a == e
		}
	case nil:
		return actual == nil
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
			defer f.Close()

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
				switch expectedCh.DataType {
				case "int32":
					data, err := ch.ReadDataInt32All()
					if err != nil {
						t.Errorf("Failed to read data: %v", err)
						continue
					}
					expectedData := toInt32Slice(t, expectedCh.Data)
					if len(data) != len(expectedData) {
						t.Errorf("Length mismatch: expected %d, got %d", len(expectedData), len(data))
					}
					for i := range data {
						if data[i] != expectedData[i] {
							t.Errorf("Data[%d] mismatch: expected %d, got %d", i, expectedData[i], data[i])
						}
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
			defer f.Close()

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

					expectedNum, _ := numScales.(float64)
					if !comparePropertyValue(prop.Value, expectedNum) {
						t.Errorf("Channel %s/%s: NI_Number_Of_Scales mismatch: expected %v, got %v",
							expectedCh.Group, expectedCh.Channel, expectedNum, prop.Value)
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
			defer f.Close()

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
				data, err := ch.ReadDataFloat64All()
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
			defer f.Close()

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
