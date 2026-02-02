package tdms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"path"
	"reflect"
	"testing"
)

func TestReadSegmentLeadIn(t *testing.T) {
	cases := []struct {
		name          string
		inputBytes    []byte
		inputFname    string
		isIndex       bool
		expectedErr   error
		expectedValue leadIn
	}{
		{
			name:        "return invalid format when TDSm magic string is wrong",
			inputBytes:  []byte{'T', 'D', 'S', 'h'},
			expectedErr: ErrInvalidFileFormat,
		},
		{
			name:        "return invalid format when TDSh magic string is wrong",
			inputBytes:  []byte{'T', 'D', 'S', 'm'},
			isIndex:     true,
			expectedErr: ErrInvalidFileFormat,
		},
		{
			name:        "return read error when input is empty",
			inputBytes:  []byte{},
			expectedErr: ErrReadFailed,
		},
		{
			name:       "read 01_minimal.tdms",
			inputFname: "01_minimal.tdms",
			expectedValue: leadIn{
				containsMetadata:     true,
				containsRawData:      true,
				containsDAQMXRawData: false,
				isInterleaved:        false,
				byteOrder:            binary.LittleEndian,
				newObjectList:        true,
				nextSegmentOffset:    483,
				rawDataOffset:        83,
			},
		},
		{
			name:       "read 02_basic_properties.tdms",
			inputFname: "02_basic_properties.tdms",
			expectedValue: leadIn{
				containsMetadata:     true,
				containsRawData:      true,
				containsDAQMXRawData: false,
				isInterleaved:        false,
				byteOrder:            binary.LittleEndian,
				newObjectList:        true,
				nextSegmentOffset:    8439,
				rawDataOffset:        439,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var f *File
			var err error
			if c.inputBytes != nil {
				f = New(bytes.NewReader(c.inputBytes), c.isIndex)
			} else {
				f, err = Open(path.Join("testdata", c.inputFname))
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				defer f.Close()
			}

			value, err := f.readSegmentLeadIn()
			if !errors.Is(err, c.expectedErr) {
				t.Errorf("expected error %v, got %v", c.expectedErr, err)
			} else if c.expectedErr == nil && (value == nil || *value != c.expectedValue) {
				t.Errorf("expected value %v, got %v", c.expectedValue, value)
			}
		})
	}
}

func TestReadSegmentMetadata(t *testing.T) {
	cases := []struct {
		name          string
		inputBytes    []byte
		inputFname    string
		expectedErr   error
		expectedValue Metadata
		isIndex       bool
	}{
		{
			name:          "read 01_minimal.tdms",
			inputFname:    "01_minimal.tdms",
			expectedErr:   nil,
			expectedValue: Metadata{
				//
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var f *File
			var err error
			if c.inputBytes != nil {
				f = New(bytes.NewReader(c.inputBytes), c.isIndex)
			} else {
				f, err = Open(path.Join("testdata", c.inputFname))
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}

			leadIn, err := f.readSegmentLeadIn()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			value, err := f.readSegmentMetadata(leadIn, nil)
			if !errors.Is(err, c.expectedErr) {
				t.Errorf("expected error %v, got %v", c.expectedErr, err)
			} else if c.expectedErr == nil && !reflect.DeepEqual(value, &c.expectedValue) {
				if value == nil {
					t.Errorf("expected value %v, got nil", c.expectedValue)
				} else {
					t.Errorf("expected value %v, got %v", c.expectedValue, *value)
				}
			}
		})
	}
}

func TestReadMetadata(t *testing.T) {
	// TODO
}

func TestReadData(t *testing.T) {
	// TODO
}
