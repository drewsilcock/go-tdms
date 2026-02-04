package tdms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
)

const (
	// This segment contains metadata.
	tocContainsMetadata uint32 = 1 << 1

	// The objects contained in this segment are different from the objects in
	// the previous segment, meaning groups and channels need to be read anew.
	tocContainsNewObjectList uint32 = 1 << 2

	// This segment contains raw data.
	tocContainsRawData uint32 = 1 << 3

	// The data in this segment is interleaved. If the data is non-interleaved,
	// the data for each channel appears contiguously in the segment in its
	// entirely before the next channel's data is present. If the data is
	// interleaved, a single data point from each channel is present one at a
	// time in order. For example, if channel 1 produces data (1, 2, 3) and
	// channel 2 produces data (4, 5, 6), non-interleaved will produces segment
	// data [1, 2, 3, 4, 5, 6] while interleaved will produce [1, 4, 2, 5, 3,
	// 6].
	tocDataIsInterleaved uint32 = 1 << 5

	// If present, all data in this segment excluding the TOC bitmask itself is
	// big endian. This includes the rest of the lead-in, the metadata and the
	// raw data.
	tocIsBigEndian uint32 = 1 << 6

	// This segment contains DAQmx raw data.
	tocContainsDAQMXRawData uint32 = 1 << 7
)

const (
	rawIndexHeaderMatchesPreviousValue uint32 = 0x00_00_00_00
	rawIndexHeaderNoRawData            uint32 = 0xff_ff_ff_ff
	rawIndexHeaderFormatChangingScaler uint32 = 0x00_00_12_69

	// The NI docs say that this value is 0x00_00_13_6a, but npTDMS author
	// believes from their experience that this is not the correct value.
	// Certainly, it is not numerically next and is possibly a typo arising from
	// confusion around little endian vs. big endian.
	rawIndexHeaderDigitalLineScaler uint32 = 0x00_00_12_6a
)

const leadInSize uint64 = 28

var (
	tdmsMagicBytes      = []byte{'T', 'D', 'S', 'm'}
	tdmsIndexMagicBytes = []byte{'T', 'D', 'S', 'h'}

	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrReadFailed         = errors.New("failed to read data")
	ErrInvalidFileFormat  = errors.New("invalid file format")
	ErrInvalidPath        = errors.New("invalid object path")
	ErrUnsupportedType    = errors.New("unsupported data type")
)

type File struct {
	Groups       map[string]Group
	Properties   map[string]Property
	IsIncomplete bool

	f        io.ReadSeeker
	size     int64
	isIndex  bool
	segments []segment

	// This does not hold pointers – we want these to be separate instances from
	// those held by the individual segment as we want to be able to modify this
	// independently to represent the object's properties at the top-level
	// throughout the file, instead of representing the object as it appears at
	// this point in the file.
	objects map[string]object
}

type Group struct {
	Name       string
	Channels   map[string]Channel
	Properties map[string]Property

	f *File
}

type Property struct {
	Name     string
	TypeCode DataType
	Value    any
}

type Channel struct {
	Name       string
	GroupName  string
	DataType   DataType
	Properties map[string]Property

	f *File
	path string
}

type leadIn struct {
	containsMetadata     bool
	containsRawData      bool
	containsDAQMXRawData bool
	isInterleaved        bool
	byteOrder            binary.ByteOrder
	newObjectList        bool
	nextSegmentOffset    uint64
	rawDataOffset        uint64
}

type metadata struct {
	// The order of objects is essential for reading the data because the data
	// is present in the same order as the objects that they correspond to.
	objectMap  map[string]*object
	objectList []*object
}

type daqmxScalerType int

const (
	daqmxScalerTypeNone daqmxScalerType = iota
	daqmxScalerTypeFormatChanging
	daqmxScalerTypeDigitalLine
)

type object struct {
	path         string
	rawDataIndex *rawDataIndex
	isDaqmxIndex bool
	properties   map[string]Property
	hasRawData   bool
}

type rawDataIndex struct {
	scaler    daqmxScalerType
	dataType  DataType
	numValues uint64

	// Only stored for variable length data types, e.g. strings, and not stored
	// for DAQmx raw data index.
	totalSize uint64

	// These are only stored for DAQmx raw data indexes.
	scalers []daqmxScaler
}

type daqmxScaler struct {
	dataType DataType

	// The documentation is very unclear about what these values actually mean.
	// It seems clear that "rawBufferIndex" here means index in the i, j way
	// instead of the raw data index, which contains metadata about the data
	// positioning, type, etc.
	rawBufferIndex            uint32
	rawByteOffsetWithinStride uint32
	sampleFormatBitmap        uint32
	scaleID                   uint32
}

type segment struct {
	offset   int64
	leadIn   *leadIn
	metadata *metadata
}

func New(reader io.ReadSeeker, isIndex bool, size int64) (*File, error) {
	// Properties can be overwritten from one segment to the next, so in order
	// to know the objects and properties, we need to read the metadata for each
	// segment upfront. For ease of use, we do this here.
	f := &File{
		Groups:     make(map[string]Group),
		Properties: make(map[string]Property),
		f:          reader,
		size:       size,
		isIndex:    isIndex,
		objects:    make(map[string]object),
	}

	if err := f.readMetadata(); err != nil {
		return nil, err
	}

	return f, nil
}

func Open(filename string) (*File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to get file info for %s: %w", filename, err)
	}

	f, err := New(
		file,
		strings.HasSuffix(filename, ".tdms_index"),
		fileInfo.Size(),
	)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return f, nil
}

func (t *File) Close() error {
	if file, ok := t.f.(*os.File); ok && file != nil {
		return file.Close()
	}

	return nil
}

// readSegmentLeadIn reads the "lead in" data for a segment, which contains
// flags telling you how to read the rest of the segment. We need the previous
// segment because certain metadata is "carried over" from one segment to the
// next, like objects and indices.
func (t *File) readSegmentLeadIn() (*leadIn, error) {
	leadInBytes := make([]byte, leadInSize)
	if _, err := t.f.Read(leadInBytes); err != nil {
		return nil, errors.Join(ErrReadFailed, err)
	}

	magicBytes := leadInBytes[:4]
	if t.isIndex {
		if !bytes.Equal(magicBytes, tdmsIndexMagicBytes) {
			return nil, errors.Join(ErrInvalidFileFormat, errors.New("invalid TDSM index magic bytes"))
		}
	} else if !bytes.Equal(magicBytes, tdmsMagicBytes) {
		return nil, errors.Join(ErrInvalidFileFormat, errors.New("invalid TDSM magic bytes"))
	}

	leadIn := leadIn{
		containsMetadata:     false,
		containsRawData:      false,
		containsDAQMXRawData: false,
		isInterleaved:        false,
		byteOrder:            binary.LittleEndian,
		newObjectList:        false,
		nextSegmentOffset:    0,
		rawDataOffset:        0,
	}

	// TOC bitmask is always little endian, even if it contains the flag
	// indicating the rest of the segment is big endian.
	tocMask := binary.LittleEndian.Uint32(leadInBytes[4:])

	if tocMask&tocContainsMetadata != 0 {
		leadIn.containsMetadata = true
	}
	if tocMask&tocContainsRawData != 0 {
		leadIn.containsRawData = true
	}
	if tocMask&tocContainsDAQMXRawData != 0 {
		leadIn.containsDAQMXRawData = true
	}
	if tocMask&tocDataIsInterleaved != 0 {
		leadIn.isInterleaved = true
	}
	if tocMask&tocIsBigEndian != 0 {
		leadIn.byteOrder = binary.BigEndian
	}
	if tocMask&tocContainsNewObjectList != 0 {
		leadIn.newObjectList = true
	}

	version := leadIn.byteOrder.Uint32(leadInBytes[8:])
	if version != 4712 && version != 4713 {
		return nil, ErrUnsupportedVersion
	}

	leadIn.nextSegmentOffset = leadIn.byteOrder.Uint64(leadInBytes[12:])
	leadIn.rawDataOffset = leadIn.byteOrder.Uint64(leadInBytes[20:])

	return &leadIn, nil
}

// readMetadata reads the metadata for each segment in the file.
func (t *File) readMetadata() error {
	t.segments = make([]segment, 0)

	var prevSegment *segment
	i := 0
	currentOffset := int64(0)

	_, err := t.f.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to beginning of metadata file: %w", err)
	}

	for {
		leadIn, err := t.readSegmentLeadIn()
		if err != nil {
			return fmt.Errorf("failed to read segment %d lead in: %w", i, err)
		}

		if leadIn.containsMetadata {
			metadata, err := t.readSegmentMetadata(leadIn, prevSegment)
			if err != nil {
				return fmt.Errorf("failed to read segment %d metadata: %w", i, err)
			}

			prevSegment = &segment{
				offset:   currentOffset,
				leadIn:   leadIn,
				metadata: metadata,
			}
			t.segments = append(t.segments, *prevSegment)
		}

		// The next segment offset is the offset from the end of the lead in.
		currentOffset += int64(leadIn.nextSegmentOffset) + int64(leadInSize)

		if leadIn.nextSegmentOffset == 0xFFFFFFFFFFFFFFFF {
			// Special value indicates that LabVIEW crashes while writing the final segment.
			t.IsIncomplete = true
			break
		}

		if currentOffset >= t.size {
			// We've reached the end of the file, all segments are read.
			t.IsIncomplete = false
			break
		}

		// If we're reading an index file, there's no data so one segment's
		// metadata leads directly into the next segment's lead in.
		if !t.isIndex {
			_, err := t.f.Seek(int64(leadIn.nextSegmentOffset), io.SeekCurrent)
			if err != nil {
				return fmt.Errorf("failed to seek to segment %d: %w", i, err)
			}
		}
	}

	// Now that we have all the channels, parse the object paths and fill the
	// file, group, and channel fields accordingly.

	// We hold the channels in a list and add them all to their respective
	// groups at the end, to avoid processing a channel before we've added the
	// corresponding group.
	channels := make(map[string]Channel, len(t.objects))

	for _, obj := range t.objects {
		groupName, channelName, err := parsePath(obj.path)
		if err != nil {
			return fmt.Errorf("failed to parse path for object %s: %w", obj.path, err)
		}

		if groupName == "" {
			// This is a root-level object, so merge the properties into the
			// root file object.
			maps.Copy(t.Properties, obj.properties)
		} else if channelName == "" {
			// This is a group object, so add it to the file's groups.
			t.Groups[groupName] = Group{
				Name:       groupName,
				Properties: obj.properties,
				Channels:   make(map[string]Channel),
				f:          t,
			}
		} else {
			// This is a channel object, so add it to the group's channels.
			channels[channelName] = Channel{
				Name:       channelName,
				GroupName:  groupName,
				DataType:   obj.rawDataIndex.dataType,
				Properties: obj.properties,
				f:          t,
				path: obj.path,
			}
		}
	}

	for channelName, channel := range channels {
		if _, exists := t.Groups[channel.GroupName]; !exists {
			return fmt.Errorf("%w: channel %s sits under non-existent group %s",
				ErrInvalidFileFormat,
				channelName,
				channel.GroupName,
			)
		}

		t.Groups[channel.GroupName].Channels[channelName] = channel
	}

	return nil
}

func (t *File) readSegmentMetadata(leadIn *leadIn, prevSegment *segment) (*metadata, error) {
	numObjects, err := readUint32(t.f, leadIn.byteOrder)
	if err != nil {
		return nil, err
	}

	objectList := make([]*object, 0, numObjects)
	objectMap := make(map[string]*object, numObjects)

	if !leadIn.newObjectList {
		if prevSegment == nil {
			return nil, errors.Join(
				ErrInvalidFileFormat,
				errors.New("lead in does not have new object list, but not prior segment"),
			)
		}

		for _, existingObj := range prevSegment.metadata.objectList {
			// We may want to update this object without changing the values in the previous segment.
			obj := *existingObj
			objectList = append(objectList, &obj)
		}
	}

	for i := 0; i < int(numObjects); i++ {
		obj, err := t.readObject(leadIn, prevSegment)
		if err != nil {
			return nil, fmt.Errorf("error reading object %d: %w", i, err)
		}

		// If a TDMS file is malformatted by having multiple objects with the
		// same path, this will overwrite the object with the last value in the
		// metadata. This is acceptable as this would be against the spec
		// anyways.
		if existingObj, ok := objectMap[obj.path]; ok {
			// At the root level, an object has raw data if it has raw data for
			// any segment. This is one example of why we can't simply use a
			// single object instance for both root-level and segment-level
			// objects.
			existingObj.hasRawData = obj.hasRawData
			existingObj.isDaqmxIndex = obj.isDaqmxIndex

			// If new object has no raw data, we keep the raw data index from
			// the previous segment.
			if obj.hasRawData {
				existingObj.rawDataIndex = obj.rawDataIndex
			}

			// New properties get added to the map while existing properties get
			// updated; properties not mentioned in the latest segment are
			// unchanged.
			maps.Copy(existingObj.properties, obj.properties)
		} else {
			// You can still add new objects to the list without the new
			// object list flag.
			objectList = append(objectList, obj)
			objectMap[obj.path] = obj
		}

		// If this object already exists in the file's collection of properties
		// (which may happen even if new object list is set or the previous
		// segment doesn't have the object because it itself has the new object
		// list flag set), we update the file's objects so that we have an up-to-date
		// list of objects. We need to merge properties but replace raw
		// data index.
		if existingObj, ok := t.objects[obj.path]; ok {
			// At the top-level, if any segment has raw data for the object,
			// then the object has raw data.
			existingObj.hasRawData = existingObj.hasRawData || obj.hasRawData

			if obj.rawDataIndex != nil {
				// It's OK to use the same pointer here because we only replace
				// the index, not update it.
				existingObj.rawDataIndex = obj.rawDataIndex

				existingObj.isDaqmxIndex = obj.isDaqmxIndex
			}

			maps.Copy(existingObj.properties, obj.properties)

			// Root level objects map has structs, not pointers, so we need to
			// remember to update the map once we've updated the fields.
			t.objects[obj.path] = existingObj
		} else {
			// File doesn't have this object yet – better add it.
			rootObj := *obj

			// We don't want to re-use the map, as above does only a shallow copy.
			rootObj.properties = make(map[string]Property, len(obj.properties))
			maps.Copy(rootObj.properties, obj.properties)

			t.objects[obj.path] = rootObj
		}
	}

	return &metadata{objectList: objectList, objectMap: objectMap}, nil
}

func (t *File) readObject(leadIn *leadIn, prevSegment *segment) (*object, error) {
	obj := object{}
	var err error

	obj.path, err = readString(t.f, leadIn.byteOrder)
	if err != nil {
		return nil, err
	}

	rawDataIndexHeader, err := readUint32(t.f, leadIn.byteOrder)
	if err != nil {
		return nil, err
	}

	rawDataIndexPresent := false

	switch rawDataIndexHeader {
	case rawIndexHeaderNoRawData:
		obj.hasRawData = false
		rawDataIndexPresent = false
	case rawIndexHeaderMatchesPreviousValue:
		// TODO: Should we be checking the prior segment or the whole file?
		if existingObj, ok := prevSegment.metadata.objectMap[obj.path]; ok {
			obj.hasRawData = existingObj.hasRawData
			obj.isDaqmxIndex = existingObj.isDaqmxIndex
			obj.rawDataIndex = existingObj.rawDataIndex
		} else {
			return nil, errors.New("raw data index matches previous value but no prior object found")
		}

		rawDataIndexPresent = false
	case rawIndexHeaderFormatChangingScaler:
		obj.rawDataIndex = &rawDataIndex{scaler: daqmxScalerTypeFormatChanging}
		obj.isDaqmxIndex = true
		rawDataIndexPresent = true
	case rawIndexHeaderDigitalLineScaler:
		obj.rawDataIndex = &rawDataIndex{scaler: daqmxScalerTypeDigitalLine}
		obj.isDaqmxIndex = true
		rawDataIndexPresent = true
	default:
		// Value is the length of the raw data index. This value seems pointless
		// as the raw data index at this point is always 20 = 0x14 bytes in
		// length (including the header). I guess it's just to differentiate it
		// from the special values above, although it seems they should've then
		// used a special value to indicate "this is a normal raw data index".
		// It's probably historical.
		obj.rawDataIndex = &rawDataIndex{scaler: daqmxScalerTypeNone}
		obj.hasRawData = true
		obj.isDaqmxIndex = false
		rawDataIndexPresent = true
	}

	if rawDataIndexPresent {
		// The normal index is always 16 bytes long so just read it all at once.
		rawDataIndexBytes := make([]byte, 16)
		if _, err := t.f.Read(rawDataIndexBytes); err != nil {
			return nil, errors.Join(ErrReadFailed, err)
		}

		obj.rawDataIndex.dataType = DataType(leadIn.byteOrder.Uint32(rawDataIndexBytes))

		dimension := leadIn.byteOrder.Uint32(rawDataIndexBytes[4:8])
		if dimension != 1 {
			return nil, errors.Join(
				ErrInvalidFileFormat,
				errors.New("in TDMS v2 raw data index dimension must be 1"),
			)
		}

		obj.rawDataIndex.numValues = leadIn.byteOrder.Uint64(rawDataIndexBytes[8:16])

		if obj.isDaqmxIndex {
			numScalers, err := readUint32(t.f, leadIn.byteOrder)
			if err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			obj.rawDataIndex.scalers = make([]daqmxScaler, numScalers)

			for i := range numScalers {
				scalerBytes := make([]byte, 16)
				if _, err := t.f.Read(scalerBytes); err != nil {
					return nil, errors.Join(ErrReadFailed, err)
				}

				scaler := &obj.rawDataIndex.scalers[i]
				scaler.dataType = DataType(leadIn.byteOrder.Uint32(scalerBytes))
				scaler.rawBufferIndex = leadIn.byteOrder.Uint32(scalerBytes[4:8])
				scaler.rawByteOffsetWithinStride = leadIn.byteOrder.Uint32(scalerBytes[8:12])
				scaler.sampleFormatBitmap = leadIn.byteOrder.Uint32(scalerBytes[12:16])
				scaler.scaleID = leadIn.byteOrder.Uint32(scalerBytes[16:20])
			}
		} else {
			// The total size is only present when the data size is variable, e.g. is a string. I can't see any other variable size data types, although I am not sure about FixedPointer and DAQmx data.
			if obj.rawDataIndex.dataType == DataTypeString {
				obj.rawDataIndex.totalSize, err = readUint64(t.f, leadIn.byteOrder)
				if err != nil {
					return nil, errors.Join(ErrReadFailed, err)
				}
			}
		}
	}

	numProps, err := readUint32(t.f, leadIn.byteOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to read number of properties: %w", err)
	}

	obj.properties = make(map[string]Property, numProps)
	for range numProps {
		propName, err := readString(t.f, leadIn.byteOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to read property name: %w", err)
		}

		propDataTypeInt, err := readUint32(t.f, leadIn.byteOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to read property data type: %w", err)
		}

		propDataType := DataType(propDataTypeInt)

		value, err := readValue(propDataType, t.f, leadIn.byteOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to read property value: %w", err)
		}

		prop := Property{
			Name:     propName,
			TypeCode: propDataType,
			Value:    value,
		}

		obj.properties[propName] = prop
	}

	return &obj, nil
}

func (c *Channel) Parent() Group {
	return c.f.Groups[c.GroupName]
}

func (c *Channel) ReadDataRaw(yield func([]byte) error) error {
	// We could keep track of which segments hold data for which channels to
	// potentially speed this up, but I don't think it would make a big
	// difference.
	for _, segment := range c.f.segments {
		segmentObject, exists := segment.metadata.objectMap[c.path]
		if !exists || !segmentObject.hasRawData {
			// A channel doesn't have to exist or have data in all segments.
			continue
		}

		channelIdx := 0
		i := 0

		// TODO: This could be sped up by keeping not an ordered list of objects
		// but a map from path to index, where we would have to pre-calculate
		// the index.
		for _, obj := range segment.metadata.objectList {
			if obj.path == c.path {
				channelIdx = i
			}

			if obj.hasRawData {
				i++
			}
		}

		if segment.leadIn.isInterleaved {
			// TODO: Handle interleaved data
		} else {
			// TODO: Handle non-interleaved data
		}
	}

	if c.rawDataIndex.totalSize > maxBufferSize {
		return ErrBufferSizeExceeded
	}

	buf := make([]byte, c.rawDataIndex.totalSize)
	if _, err := io.ReadFull(c.f, buf); err != nil {
		return fmt.Errorf("failed to read raw data: %w", err)
	}

	return yield(buf)
