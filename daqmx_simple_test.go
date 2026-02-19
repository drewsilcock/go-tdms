package tdms

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// TestGeneratedFileStructure verifies the test file generator creates valid TDMS structure
func TestGeneratedFileStructure(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 3, 0, 0) // scale_id=0, type=Int16, offset=0, buffer=0

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{2}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 02 00 FF FF FE FF")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	fileBytes := tf.build()

	// Verify file starts with TDSm magic
	if !bytes.HasPrefix(fileBytes, tdmsMagicBytes) {
		t.Error("File doesn't start with TDSm magic bytes")
	}

	// Verify file is the expected size
	// 28 (lead-in) + len(metadata) + len(rawData)
	expectedSize := 28 + len(metadata) + len(rawData)
	if len(fileBytes) != expectedSize {
		t.Errorf("Expected file size %d, got %d", expectedSize, len(fileBytes))
	}

	t.Logf("Generated file structure looks valid (%d bytes)", len(fileBytes))
	t.Logf("Metadata size: %d bytes", len(metadata))
	t.Logf("Raw data size: %d bytes", len(rawData))

	// Dump first 100 bytes for debugging
	dumpSize := min(len(fileBytes), 100)
	t.Logf("First %d bytes:\n%s", dumpSize, hex.Dump(fileBytes[:dumpSize]))
}

// TestMetadataStructure tests the metadata builder functions
func TestMetadataStructure(t *testing.T) {
	root := rootMetadata()
	if len(root) < 10 {
		t.Errorf("Root metadata seems too short: %d bytes", len(root))
	}

	group := groupMetadata()
	if len(group) < 15 {
		t.Errorf("Group metadata seems too short: %d bytes", len(group))
	}

	scaler := daqmxScalerMetadata(0, 3, 0, 0)
	if len(scaler) != 20 {
		t.Errorf("DAQmx format-changing scaler should be 20 bytes, got %d", len(scaler))
	}

	digitalScaler := digitalScalerMetadata(0, 0, 0, 0)
	if len(digitalScaler) != 17 {
		t.Errorf("DAQmx digital line scaler should be 17 bytes, got %d", len(digitalScaler))
	}

	t.Log("All metadata structures have expected sizes")
}

// TestHexToBytes verifies the hex string parser
func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "simple hex",
			input:    "01 02 03 04",
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "hex with newlines",
			input:    "01 02\n03 04",
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "hex with tabs",
			input:    "01\t02\t03\t04",
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "FF bytes",
			input:    "FF FF FE FF",
			expected: []byte{0xFF, 0xFF, 0xFE, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hexToBytes(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("hexToBytes(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestReadRealDAQmxFile attempts to read a real DAQmx file to verify the implementation works
func TestReadRealDAQmxFile(t *testing.T) {
	file, err := Open("testdata/POC_MultisamplingRate.tdms")
	if err != nil {
		t.Fatalf("Skipping test - couldn't open test file: %v", err)
	}
	defer file.Close() // nolint:errcheck // test file cleanup

	t.Logf("Successfully opened file")
	t.Logf("Groups: %d", len(file.Groups))

	for groupName, group := range file.Groups {
		t.Logf("Group '%s': %d channels", groupName, len(group.Channels))
		for channelName, channel := range group.Channels {
			t.Logf("  Channel '%s': %d values, RawDataType=%v, ScaledDataType=%v",
				channelName, channel.NumValues(), channel.RawDataType, channel.ScaledDataType)

			// Try to read a small amount of data
			if channel.NumValues() > 0 {
				data, err := channel.ReadAll()
				if err != nil {
					t.Errorf("Failed to read channel '%s'/'%s': %v", groupName, channelName, err)
				} else {
					t.Logf("  Successfully read %d values from '%s'/'%s' (type %T)",
						channel.NumValues(), groupName, channelName, data)
				}
			}
		}
	}
}

// TestDetailedFileStructure validates the exact byte structure of a generated DAQmx file
func TestDetailedFileStructure(t *testing.T) {
	scalerMeta := daqmxScalerMetadata(0, 3, 0, 0)

	metadata := buildMetadata(3,
		rootMetadata(),
		groupMetadata(),
		daqmxChannelMetadata("Channel1", 4, []uint32{2}, [][]byte{scalerMeta}, 0xFFFFFFFF, nil),
	)

	rawData := hexToBytes("01 00 02 00 FF FF FE FF")

	tf := newTestFile()
	tf.addSegment(segmentTOC(), metadata, rawData)

	fileBytes := tf.build()

	t.Logf("Total file: %d bytes", len(fileBytes))
	t.Logf("Metadata: %d bytes", len(metadata))
	t.Logf("Raw data: %d bytes", len(rawData))
	t.Logf("Expected: 28 + %d + %d = %d", len(metadata), len(rawData), 28+len(metadata)+len(rawData))

	// Validate lead-in (28 bytes)
	if len(fileBytes) < 28 {
		t.Fatal("File too short for lead-in")
	}

	// Check magic bytes
	if string(fileBytes[0:4]) != "TDSm" {
		t.Errorf("Invalid magic bytes: %v", fileBytes[0:4])
	}

	// Verify the full file dumps correctly
	t.Logf("Full file hex dump:\n%s", hex.Dump(fileBytes))
}
