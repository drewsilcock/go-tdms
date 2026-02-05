package tdms

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// For the moment, our tests are just parsing a whole file and checking that the
// entire parsed file is exactly what we expect it to be. Having separate
// smaller tests that are testing each individual segment makes it easier to
// find where the failure is happening and explore more failure cases.
//
// Currently, we aren't testing any failure modes to ensure the code fails for
// the right reason and in the right place and way. To do this, we can modify
// some of the existing correct files to introduce deliberate errors in precise
// locations, which we can then test for. For ease of use, we can add this
// functionality into the generate test files script.

type testCase struct {
	filename string
	expected *File
	data     map[string][]any
	err      error
}

var testCases []testCase = buildTestCases()

var cmpOptions = []cmp.Option{
	cmp.AllowUnexported(
		File{}, Group{}, Channel{}, Property{}, segment{},
		leadIn{}, metadata{}, object{}, objectIndex{}, dataChunk{},
	),
	cmpopts.IgnoreTypes(os.File{}),
	cmpopts.IgnoreFields(Group{}, "f"),
	cmpopts.IgnoreFields(Channel{}, "f"),
}

func buildTestCases() []testCase {
	testCases := []testCase{
		{
			filename: "01_minimal.tdms",
			expected: &File{
				Groups: map[string]Group{
					"Group": {
						Name: "Group",
						Channels: map[string]Channel{
							"Channel": {
								Name:       "Channel",
								GroupName:  "Group",
								Properties: map[string]Property{},
								DataType:   DataTypeInt32,
								f:          nil,
								path:       "/'Group'/'Channel'",
								dataChunks: []dataChunk{
									{
										offset:    111,
										order:     binary.LittleEndian,
										size:      400,
										numValues: 100,
									},
								},
							},
						},
						Properties: map[string]Property{},
					},
				},
				Properties:   map[string]Property{},
				IsIncomplete: false,
				size:         511,
				isIndex:      false,
				segments: []segment{
					{
						offset: 0,
						leadIn: &leadIn{
							containsMetadata:     true,
							containsRawData:      true,
							containsDAQMXRawData: false,
							isInterleaved:        false,
							byteOrder:            binary.LittleEndian,
							newObjectList:        true,
							nextSegmentOffset:    483,
							rawDataOffset:        83,
						},
						metadata: &metadata{
							objectOrder: []string{"/", "/'Group'", "/'Group'/'Channel'"},
							objects: map[string]object{
								"/": {
									path:       "/",
									index:      nil,
									properties: make(map[string]Property),
								},
								"/'Group'": {
									path:       "/'Group'",
									index:      nil,
									properties: make(map[string]Property),
								},
								"/'Group'/'Channel'": {
									path: "/'Group'/'Channel'",
									index: &objectIndex{
										scalerType: daqmxScalerTypeNone,
										dataType:   DataTypeInt32,
										numValues:  100,
										totalSize:  400,
										offset:     111,
										stride:     0, // No other channels
										scalers:    nil,
										widths:     nil,
									},
									properties: make(map[string]Property),
								},
							},
							chunkSize: 400,
							numChunks: 1,
						},
					},
				},
				objects: map[string]object{
					"/": {
						path:       "/",
						index:      nil,
						properties: make(map[string]Property),
					},
					"/'Group'": {
						path:       "/'Group'",
						index:      nil,
						properties: make(map[string]Property),
					},
					"/'Group'/'Channel'": {
						path: "/'Group'/'Channel'",
						index: &objectIndex{
							scalerType: daqmxScalerTypeNone,
							dataType:   DataTypeInt32,
							numValues:  100,
							totalSize:  400,
							offset:     111,
							stride:     0,
							scalers:    nil,
							widths:     nil,
						},
						properties: make(map[string]Property),
					},
				},
			},

			data: map[string][]any{
				"Channel": []any{},
			},
		},

		//{
		//	filename: "02_basic_properties.tdms",
		//	expected: &File{
		//		//
		//	},
		//},
	}

	testValues := make([]any, 100)
	for i := range 100 {
		testValues[i] = i
	}
	testCases[0].data["Channel"] = testValues

	return testCases
}

func TestReadTDMSFiles(t *testing.T) {
	for _, c := range testCases {
		t.Run(fmt.Sprintf("read %s", c.filename), func(t *testing.T) {
			f, err := Open(path.Join("testdata", c.filename))
			if !errors.Is(err, c.err) {
				t.Errorf("expected error %v, got %v", c.err, err)
				return
			}
			defer f.Close()

			// We don't want to test the equality of the os.File object.
			c.expected.f = f.f

			if !cmp.Equal(c.expected, f, cmpOptions...) {
				t.Errorf("expected value differs from actual value:\n\n %s", diff(c.expected, f, cmpOptions...))
				return
			}

			ch := f.Groups["Group"].Channels["Channel"]
			ch.ReadDataAsUint32()
		})
	}
}

func diff(want, got any, opts ...cmp.Option) string {
	diff := cmp.Diff(want, got, opts...)
	if diff == "" {
		return ""
	}

	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "-"):
			lines[i] = "\x1b[31m" + line + "\x1b[0m" // Red for removals
		case strings.HasPrefix(line, "+"):
			lines[i] = "\x1b[32m" + line + "\x1b[0m" // Green for additions
		}
	}
	return strings.Join(lines, "\n")
}
