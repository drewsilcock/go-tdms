package tdms

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
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
	err      error
}

var testCases []testCase = buildTestCases()

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
							},
						},
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
						metadata: buildMetadata([]object{
							{
								path:         "/",
								hasRawData:   false,
								rawDataIndex: nil,
								isDaqmxIndex: false,
								properties:   make(map[string]Property),
							},
							{
								path:         "/'Group'",
								hasRawData:   false,
								rawDataIndex: nil,
								isDaqmxIndex: false,
								properties:   make(map[string]Property),
							},
							{
								path:       "/'Group'/'Channel'",
								hasRawData: true,
								rawDataIndex: &rawDataIndex{
									scaler:    daqmxScalerTypeNone,
									dataType:  DataTypeInt32,
									numValues: 100,
									totalSize: 0,
									scalers:   nil,
								},
								isDaqmxIndex: false,
								properties:   make(map[string]Property),
							},
						}),
					},
				},
				objects: map[string]object{
					"/": {
						path:         "/",
						hasRawData:   false,
						rawDataIndex: nil,
						isDaqmxIndex: false,
						properties:   make(map[string]Property),
					},
					"/'Group'": {
						path:         "/'Group'",
						hasRawData:   false,
						rawDataIndex: nil,
						isDaqmxIndex: false,
						properties:   make(map[string]Property),
					},
					"/'Group'/'Channel'": {
						path:       "/'Group'/'Channel'",
						hasRawData: true,
						rawDataIndex: &rawDataIndex{
							scaler:    daqmxScalerTypeNone,
							dataType:  DataTypeInt32,
							numValues: 100,
							totalSize: 0,
							scalers:   nil,
						},
						isDaqmxIndex: false,
						properties:   make(map[string]Property),
					},
				},
			},
		},

		{
			filename: "02_basic_properties.tdms",
			expected: &File{
				//
			},
		},
	}

	// Now update the internal pointer references. Note: we can't really set the
	// ReadSeeker on the root file object so we need to just set that to null
	// before we do the test comparison.
	for i := range testCases {
		for groupName, group := range testCases[i].expected.Groups {
			for channelName, channel := range group.Channels {
				channel.f = testCases[i].expected
				group.Channels[channelName] = channel
			}

			group.f = testCases[i].expected
			testCases[i].expected.Groups[groupName] = group
		}
	}

	return testCases
}

func buildMetadata(objects []object) *metadata {
	m := metadata{
		objectMap:  make(map[string]*object, len(objects)),
		objectList: make([]*object, len(objects)),
	}

	for i, obj := range objects {
		m.objectMap[obj.path] = &obj
		m.objectList[i] = &obj
	}

	return &m
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

			if !reflect.DeepEqual(c.expected, f) {
				t.Errorf("expected value differs from actual value:\n\n %s", formatDiffs(deepDiff(c.expected, f)))
				return
			}

			// TODO: Test reading data from channels.
		})
	}
}

type diff struct {
	path     string
	expected any
	actual   any
	message  string
}

func deepDiff(expected, actual any) []diff {
	var diffs []diff
	diffRecursive(expected, actual, "root", &diffs)
	return diffs
}

func diffRecursive(expected, actual any, path string, diffs *[]diff) {
	expVal := reflect.ValueOf(expected)
	actVal := reflect.ValueOf(actual)

	// Recursively dereference pointers so that we're comparing values to values
	// (or nils to nils).
	for expVal.Kind() == reflect.Pointer {
		if expVal.IsNil() {
			break
		}
		expVal = expVal.Elem()
	}
	for actVal.Kind() == reflect.Pointer {
		if actVal.IsNil() {
			break
		}
		actVal = actVal.Elem()
	}

	if !expVal.IsValid() && !actVal.IsValid() {
		return
	}

	if !expVal.IsValid() || !actVal.IsValid() {
		*diffs = append(*diffs, diff{
			path:     path,
			expected: expected,
			actual:   actual,
			message:  "value is nil",
		})
		return
	}

	// If types differ, report and stop
	if expVal.Type() != actVal.Type() {
		*diffs = append(*diffs, diff{
			path:     path,
			expected: expected,
			actual:   actual,
			message:  fmt.Sprintf("type mismatch: expected %s, got %s", expVal.Type(), actVal.Type()),
		})
		return
	}

	switch expVal.Kind() {
	case reflect.Struct:
		for i := 0; i < expVal.NumField(); i++ {
			field := expVal.Type().Field(i)
			newPath := path + "." + field.Name
			diffRecursive(expVal.Field(i).Interface(), actVal.Field(i).Interface(), newPath, diffs)
		}

	case reflect.Slice, reflect.Array:
		if expVal.Len() != actVal.Len() {
			*diffs = append(*diffs, diff{
				path:     path,
				expected: expVal.Len(),
				actual:   actVal.Len(),
				message:  fmt.Sprintf("length mismatch: expected %d, got %d", expVal.Len(), actVal.Len()),
			})
		}
		for i := 0; i < expVal.Len() && i < actVal.Len(); i++ {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			diffRecursive(expVal.Index(i).Interface(), actVal.Index(i).Interface(), newPath, diffs)
		}

	case reflect.Map:
		// Check all keys in expected
		for _, key := range expVal.MapKeys() {
			newPath := fmt.Sprintf("%s[%v]", path, key.Interface())
			expMapVal := expVal.MapIndex(key)
			actMapVal := actVal.MapIndex(key)

			if !actMapVal.IsValid() {
				*diffs = append(*diffs, diff{
					path:     newPath,
					expected: expMapVal.Interface(),
					actual:   nil,
					message:  "missing key",
				})
			} else {
				diffRecursive(expMapVal.Interface(), actMapVal.Interface(), newPath, diffs)
			}
		}
		// Check for extra keys in actual
		for _, key := range actVal.MapKeys() {
			if !expVal.MapIndex(key).IsValid() {
				newPath := fmt.Sprintf("%s[%v]", path, key.Interface())
				*diffs = append(*diffs, diff{
					path:     newPath,
					expected: nil,
					actual:   actVal.MapIndex(key).Interface(),
					message:  "unexpected key",
				})
			}
		}

	default:
		// Primitive types
		if expVal.Interface() != actVal.Interface() {
			*diffs = append(*diffs, diff{
				path:     path,
				expected: expected,
				actual:   actual,
				message:  fmt.Sprintf("expected %v, got %v", expected, actual),
			})
		}
	}
}

func formatDiffs(diffs []diff) string {
	if len(diffs) == 0 {
		return "No differences"
	}

	// Sort by path for consistent output
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].path < diffs[j].path
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d difference(s):\n\n", len(diffs)))

	for _, diff := range diffs {
		sb.WriteString(fmt.Sprintf("  %s\n", diff.path))
		sb.WriteString(fmt.Sprintf("    Expected: %v\n", diff.expected))
		sb.WriteString(fmt.Sprintf("    Actual:   %v\n", diff.actual))
		if diff.message != "" {
			sb.WriteString(fmt.Sprintf("    (%s)\n", diff.message))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
