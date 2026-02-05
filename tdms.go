package tdms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"os"
	"strings"
	"time"
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

const segmentIncomplete uint64 = 0xff_ff_ff_ff_ff_ff_ff_ff

const (
	leadInSize uint64 = 28
	scalerSize uint32 = 16
)

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

	f              *File
	path           string
	dataChunks     []dataChunk
	totalNumValues uint64
}

type segment struct {
	offset   int64
	leadIn   *leadIn
	metadata *metadata
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
	objects map[string]object

	// The order of objects is essential for reading the data because the data
	// is present in the same order as the objects that they correspond to.
	objectOrder []string

	// Segments can contain multiple chunks of data; where the lead in/metadata
	// of the segment remains unchanged, you can simply write additional chunks
	// of data (either interleaved or non-interleaved) one after the other.
	numChunks uint64
	chunkSize uint64
}

type daqmxScalerType int

const (
	daqmxScalerTypeNone daqmxScalerType = iota
	daqmxScalerTypeFormatChanging
	daqmxScalerTypeDigitalLine
)

type object struct {
	path string

	// If index is nil, that means there's no raw data for this object.
	index      *objectIndex
	properties map[string]Property
}

type objectIndex struct {
	// If scaler type is none, that means this is not DAQmx data. Otherwise, it
	// is.
	scalerType daqmxScalerType
	dataType   DataType
	numValues  uint64

	// For variable-size data types, e.g. strings, this is taken from the file
	// itself. Otherwise, it is calculated from data type size and number of
	// values. This refers to the total size of this channel in bytes for a
	// single chunk.
	totalSize uint64

	// Only stored for DAQmx raw data.
	scalers []daqmxScaler

	// Only stored for DAQmx raw data.
	widths []uint32

	// Offset is the absolute offset from the beginning of the file.
	offset int64

	// Stride is the distance from one data point to the next, when the data is
	// interleaved. It is equal to the size of a single datum for all objects
	// other than the current object.
	stride int64
}

// dataChunk is similar to objectIndex, but is a single object index can
// correspond to multiple chunks whereas a single dataChunk instance corresponds
// to a single raw data chunk in the TDMS file.
//
// Note that a dataChunk instance is specific to an individual object, meaning a
// segment in a TDMS file with 2 channels and 3 chunks will have 6 dataChunk
// instances corresponding to it.
//
// This is purely for ease of use
// to make reading simpler and to keep all the necessary information self-contained.
type dataChunk struct {
	// offset is absolute from the start of the file
	offset        int64
	isInterleaved bool
	order         binary.ByteOrder
	size          uint64
	numValues     uint64
	stride        int64
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
			metadata, err := t.readSegmentMetadata(currentOffset, leadIn, prevSegment)
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

		if leadIn.nextSegmentOffset == segmentIncomplete {
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
			_, err := t.f.Seek(currentOffset, io.SeekStart)
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

			// Pre-compute the positions and metadata for each data chunk that
			// this channel has, if any. This makes reading data for this
			// channel much simpler.
			chunks := make([]dataChunk, 0, len(t.segments))
			for _, segment := range t.segments {
				if !segment.leadIn.containsRawData {
					continue
				}

				obj, ok := segment.metadata.objects[obj.path]
				if !ok || obj.index == nil {
					continue
				}

				for chunkIdx := range segment.metadata.numChunks {
					chunks = append(chunks, dataChunk{
						offset:        obj.index.offset + int64(chunkIdx*segment.metadata.chunkSize),
						isInterleaved: segment.leadIn.isInterleaved,
						order:         segment.leadIn.byteOrder,
						size:          obj.index.totalSize,
						numValues:     obj.index.numValues,
						stride:        obj.index.stride,
					})
				}
			}

			totalNumValues := uint64(0)
			for _, chunk := range chunks {
				totalNumValues += chunk.numValues
			}

			channels[channelName] = Channel{
				Name:           channelName,
				GroupName:      groupName,
				DataType:       obj.index.dataType,
				Properties:     obj.properties,
				f:              t,
				path:           obj.path,
				dataChunks:     chunks,
				totalNumValues: totalNumValues,
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

func (t *File) readSegmentMetadata(segmentOffset int64, leadIn *leadIn, prevSegment *segment) (*metadata, error) {
	numObjects, err := readUint32(t.f, leadIn.byteOrder)
	if err != nil {
		return nil, err
	}

	m := metadata{
		objects:     make(map[string]object, numObjects),
		objectOrder: make([]string, 0, numObjects),
	}

	if !leadIn.newObjectList {
		if prevSegment == nil {
			return nil, errors.Join(
				ErrInvalidFileFormat,
				errors.New("lead in does not have new object list, but not prior segment"),
			)
		}

		for _, existingObjPath := range prevSegment.metadata.objectOrder {
			m.objectOrder = append(m.objectOrder, existingObjPath)
			m.objects[existingObjPath] = prevSegment.metadata.objects[existingObjPath]
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
		if existingObj, ok := m.objects[obj.path]; ok {
			// If new object has no raw data, we keep the raw data index from
			// the previous segment.
			if obj.index != nil {
				existingObj.index = obj.index
			}

			// New properties get added to the map while existing properties get
			// updated; properties not mentioned in the latest segment are
			// unchanged.
			maps.Copy(existingObj.properties, obj.properties)

			m.objects[obj.path] = existingObj
		} else {
			// You can still add new objects to the list without the new
			// object list flag.
			m.objectOrder = append(m.objectOrder, obj.path)
			m.objects[obj.path] = *obj
		}

		// If this object already exists in the file's collection of properties
		// (which may happen even if new object list is set or the previous
		// segment doesn't have the object because it itself has the new object
		// list flag set), we update the file's objects so that we have an up-to-date
		// list of objects. We need to merge properties but replace raw
		// data index.
		if existingObj, ok := t.objects[obj.path]; ok {
			// At the top-level, the raw data index has very little significance
			// as it is very much segment-specific. The only useful piece of
			// information is the data type, which is forbidden from changing
			// from one segment to the next for a specific object. This sets the
			// index equal to the last non-nil value, which you can use to
			// extract data type and scalers. It's not clear if scalers can
			// change from one segment to the next, which implies we have to
			// handle this as an edge case; you should thus be using
			// segment-specific objects for that information.
			if obj.index != nil {
				// It's OK to use the same pointer here because we only replace
				// the index, not update it.
				existingObj.index = obj.index
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

	// Calculate the number of chunks based on the next segment offset and
	// the total size of each chunk.
	m.chunkSize = 0
	for _, obj := range m.objects {
		if obj.index != nil {
			m.chunkSize += obj.index.totalSize
		}
	}

	totalRawDataSize := leadIn.nextSegmentOffset - leadIn.rawDataOffset
	if leadIn.nextSegmentOffset == segmentIncomplete {
		rawDataAbsolutePosition := uint64(segmentOffset) + leadInSize + leadIn.rawDataOffset
		totalRawDataSize = uint64(t.size) - rawDataAbsolutePosition
	}

	m.numChunks = totalRawDataSize / m.chunkSize

	// Calculate the offset from the start of the segment to the first data
	// point for the object, as well as the "stride" between successive data
	// points when the data is interleaved. The stride isn't useful when the
	// data is not interleaved, but it's cheap to calculate.
	dataOffset := segmentOffset + int64(leadInSize+leadIn.rawDataOffset)
	for _, objectPath := range m.objectOrder {
		obj := m.objects[objectPath]
		if obj.index == nil || obj.index.totalSize == 0 {
			continue
		}

		obj.index.offset = dataOffset
		dataOffset += int64(obj.index.totalSize)

		obj.index.stride = int64(m.chunkSize - obj.index.totalSize)
	}

	return &m, nil
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
		obj.index = nil
		rawDataIndexPresent = false
	case rawIndexHeaderMatchesPreviousValue:
		if existingObj, ok := prevSegment.metadata.objects[obj.path]; ok {
			// We don't bother copying the index because we won't change it.
			obj.index = existingObj.index
		} else {
			return nil, errors.New("raw data index matches previous value but no prior object found")
		}

		rawDataIndexPresent = false
	case rawIndexHeaderFormatChangingScaler:
		obj.index = &objectIndex{scalerType: daqmxScalerTypeFormatChanging}
		rawDataIndexPresent = true
	case rawIndexHeaderDigitalLineScaler:
		obj.index = &objectIndex{scalerType: daqmxScalerTypeDigitalLine}
		rawDataIndexPresent = true
	default:
		// Value is the length of the raw data index. This value seems pointless
		// as the raw data index at this point is always 20 = 0x14 bytes in
		// length (including the header). I guess it's just to differentiate it
		// from the special values above, although it seems they should've then
		// used a special value to indicate "this is a normal raw data index".
		// It's probably historical.
		obj.index = &objectIndex{scalerType: daqmxScalerTypeNone}
		rawDataIndexPresent = true
	}

	if rawDataIndexPresent {
		// The normal index is always 16 bytes long so just read it all at once.
		rawDataIndexBytes := make([]byte, 16)
		if _, err := t.f.Read(rawDataIndexBytes); err != nil {
			return nil, errors.Join(ErrReadFailed, err)
		}

		obj.index.dataType = DataType(leadIn.byteOrder.Uint32(rawDataIndexBytes))

		// It is explicitly prohibited to have an interleaved segment with
		// variable-width data types.
		if obj.index.dataType == DataTypeString && leadIn.isInterleaved {
			return nil, fmt.Errorf(
				"%w: interleaved segments are not allowed with variable-width data types",
				ErrInvalidFileFormat,
			)
		}

		dimension := leadIn.byteOrder.Uint32(rawDataIndexBytes[4:8])
		if dimension != 1 {
			return nil, errors.Join(
				ErrInvalidFileFormat,
				errors.New("in TDMS v2 raw data index dimension must be 1"),
			)
		}

		obj.index.numValues = leadIn.byteOrder.Uint64(rawDataIndexBytes[8:16])

		if obj.index.scalerType == daqmxScalerTypeNone {
			// The total size is only present when the data size is variable,
			// e.g. is a string. I can't see any other variable size data types,
			// although I am not sure about FixedPointer and DAQmx data.
			if obj.index.dataType == DataTypeString {
				obj.index.totalSize, err = readUint64(t.f, leadIn.byteOrder)
				if err != nil {
					return nil, errors.Join(ErrReadFailed, err)
				}
			} else {
				obj.index.totalSize = obj.index.numValues * uint64(obj.index.dataType.Size())
			}
		} else {
			numScalers, err := readUint32(t.f, leadIn.byteOrder)
			if err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			obj.index.scalers = make([]daqmxScaler, numScalers)

			scalersBytes := make([]byte, scalerSize*numScalers)
			if _, err := t.f.Read(scalersBytes); err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			for i := range numScalers {
				scalerBytes := scalersBytes[i*scalerSize : (i+1)*scalerSize]

				scaler := &obj.index.scalers[i]
				scaler.dataType = DataType(leadIn.byteOrder.Uint32(scalerBytes))
				scaler.rawBufferIndex = leadIn.byteOrder.Uint32(scalerBytes[4:8])
				scaler.rawByteOffsetWithinStride = leadIn.byteOrder.Uint32(scalerBytes[8:12])
				scaler.sampleFormatBitmap = leadIn.byteOrder.Uint32(scalerBytes[12:16])
				scaler.scaleID = leadIn.byteOrder.Uint32(scalerBytes[16:20])
			}

			numWidths, err := readUint32(t.f, leadIn.byteOrder)
			if err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			obj.index.widths = make([]uint32, numWidths)

			widthsBytes := make([]byte, 4*numWidths)
			if _, err := t.f.Read(widthsBytes); err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			for i := range numWidths {
				widthBytes := widthsBytes[i*4:]
				obj.index.widths[i] = leadIn.byteOrder.Uint32(widthBytes)
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

func (ch *Channel) Group() Group {
	return ch.f.Groups[ch.GroupName]
}

type readOptions struct {
	batchSize int
}

type ReadOption func(*readOptions)

func BatchSize(batchSize int) ReadOption {
	return func(opts *readOptions) {
		opts.batchSize = batchSize
	}
}

// Data streaming functions that yield each item at a time.

func (ch *Channel) ReadDataAsInt8(options ...ReadOption) iter.Seq2[int8, error] {
	return StreamReader(ch, options, DataTypeInt8, interpretInt8)
}

func (ch *Channel) ReadDataAsInt16(options ...ReadOption) iter.Seq2[int16, error] {
	return StreamReader(ch, options, DataTypeInt16, interpretInt16)
}

func (ch *Channel) ReadDataAsInt32(options ...ReadOption) iter.Seq2[int32, error] {
	return StreamReader(ch, options, DataTypeInt32, interpretInt32)
}

func (ch *Channel) ReadDataAsInt64(options ...ReadOption) iter.Seq2[int64, error] {
	return StreamReader(ch, options, DataTypeInt64, interpretInt64)
}

func (ch *Channel) ReadDataAsUint8(options ...ReadOption) iter.Seq2[uint8, error] {
	return StreamReader(ch, options, DataTypeUint8, interpretUint8)
}

func (ch *Channel) ReadDataAsUint16(options ...ReadOption) iter.Seq2[uint16, error] {
	return StreamReader(ch, options, DataTypeUint16, interpretUint16)
}

func (ch *Channel) ReadDataAsUint32(options ...ReadOption) iter.Seq2[uint32, error] {
	return StreamReader(ch, options, DataTypeUint32, interpretUint32)
}

func (ch *Channel) ReadDataAsUint64(options ...ReadOption) iter.Seq2[uint64, error] {
	return StreamReader(ch, options, DataTypeUint64, interpretUint64)
}

func (ch *Channel) ReadDataAsFloat32(options ...ReadOption) iter.Seq2[float32, error] {
	return StreamReader(ch, options, DataTypeFloat32, interpretFloat32)
}

func (ch *Channel) ReadDataAsFloat64(options ...ReadOption) iter.Seq2[float64, error] {
	return StreamReader(ch, options, DataTypeFloat64, interpretFloat64)
}

func (ch *Channel) ReadDataAsString(options ...ReadOption) iter.Seq2[string, error] {
	return StreamReader(ch, options, DataTypeString, interpretString)
}

func (ch *Channel) ReadDataAsBool(options ...ReadOption) iter.Seq2[bool, error] {
	return StreamReader(ch, options, DataTypeBool, interpretBool)
}

func (ch *Channel) ReadDataAsTime(options ...ReadOption) iter.Seq2[time.Time, error] {
	return StreamReader(ch, options, DataTypeTime, interpretTime)
}

func (ch *Channel) ReadDataAsComplex64(options ...ReadOption) iter.Seq2[complex64, error] {
	return StreamReader(ch, options, DataTypeComplex64, interpretComplex64)
}

func (ch *Channel) ReadDataAsComplex128(options ...ReadOption) iter.Seq2[complex128, error] {
	return StreamReader(ch, options, DataTypeComplex128, interpretComplex128)
}

// Data streaming functions that yield items in batches.

func (ch *Channel) ReadDataAsInt8Batch(options ...ReadOption) iter.Seq2[[]int8, error] {
	return BatchStreamReader(ch, options, DataTypeInt8, interpretInt8)
}

func (ch *Channel) ReadDataAsInt16Batch(options ...ReadOption) iter.Seq2[[]int16, error] {
	return BatchStreamReader(ch, options, DataTypeInt16, interpretInt16)
}

func (ch *Channel) ReadDataAsInt32Batch(options ...ReadOption) iter.Seq2[[]int32, error] {
	return BatchStreamReader(ch, options, DataTypeInt32, interpretInt32)
}

func (ch *Channel) ReadDataAsInt64Batch(options ...ReadOption) iter.Seq2[[]int64, error] {
	return BatchStreamReader(ch, options, DataTypeInt64, interpretInt64)
}

func (ch *Channel) ReadDataAsUint8Batch(options ...ReadOption) iter.Seq2[[]uint8, error] {
	return BatchStreamReader(ch, options, DataTypeUint8, interpretUint8)
}

func (ch *Channel) ReadDataAsUint16Batch(options ...ReadOption) iter.Seq2[[]uint16, error] {
	return BatchStreamReader(ch, options, DataTypeUint16, interpretUint16)
}

func (ch *Channel) ReadDataAsUint32Batch(options ...ReadOption) iter.Seq2[[]uint32, error] {
	return BatchStreamReader(ch, options, DataTypeUint32, interpretUint32)
}

func (ch *Channel) ReadDataAsUint64Batch(options ...ReadOption) iter.Seq2[[]uint64, error] {
	return BatchStreamReader(ch, options, DataTypeUint64, interpretUint64)
}

func (ch *Channel) ReadDataAsFloat32Batch(options ...ReadOption) iter.Seq2[[]float32, error] {
	return BatchStreamReader(ch, options, DataTypeFloat32, interpretFloat32)
}

func (ch *Channel) ReadDataAsFloat64Batch(options ...ReadOption) iter.Seq2[[]float64, error] {
	return BatchStreamReader(ch, options, DataTypeFloat64, interpretFloat64)
}

func (ch *Channel) ReadDataAsFloat128Batch(options ...ReadOption) iter.Seq2[[]Float128, error] {
	return BatchStreamReader(ch, options, DataTypeFloat128, interpretFloat128)
}

func (ch *Channel) ReadDataAsStringBatch(options ...ReadOption) iter.Seq2[[]string, error] {
	return BatchStreamReader(ch, options, DataTypeString, interpretString)
}

func (ch *Channel) ReadDataAsBoolBatch(options ...ReadOption) iter.Seq2[[]bool, error] {
	return BatchStreamReader(ch, options, DataTypeBool, interpretBool)
}

func (ch *Channel) ReadDataAsTimeBatch(options ...ReadOption) iter.Seq2[[]time.Time, error] {
	return BatchStreamReader(ch, options, DataTypeTime, interpretTime)
}

func (ch *Channel) ReadDataAsComplex64Batch(options ...ReadOption) iter.Seq2[[]complex64, error] {
	return BatchStreamReader(ch, options, DataTypeComplex64, interpretComplex64)
}

func (ch *Channel) ReadDataAsComplex128Batch(options ...ReadOption) iter.Seq2[[]complex128, error] {
	return BatchStreamReader(ch, options, DataTypeComplex128, interpretComplex128)
}

// Data streaming functions that read all the whole for a channel in one go.

func (ch *Channel) ReadDataInt8All(options ...ReadOption) ([]int8, error) {
	return readAllData(ch, options, DataTypeInt8, interpretInt8)
}

func (ch *Channel) ReadDataInt16All(options ...ReadOption) ([]int16, error) {
	return readAllData(ch, options, DataTypeInt16, interpretInt16)
}

func (ch *Channel) ReadDataInt32All(options ...ReadOption) ([]int32, error) {
	return readAllData(ch, options, DataTypeInt32, interpretInt32)
}

func (ch *Channel) ReadDataInt64All(options ...ReadOption) ([]int64, error) {
	return readAllData(ch, options, DataTypeInt64, interpretInt64)
}

func (ch *Channel) ReadDataUint8All(options ...ReadOption) ([]uint8, error) {
	return readAllData(ch, options, DataTypeUint8, interpretUint8)
}

func (ch *Channel) ReadDataUint16All(options ...ReadOption) ([]uint16, error) {
	return readAllData(ch, options, DataTypeUint16, interpretUint16)
}

func (ch *Channel) ReadDataUint32All(options ...ReadOption) ([]uint32, error) {
	return readAllData(ch, options, DataTypeUint32, interpretUint32)
}

func (ch *Channel) ReadDataUint64All(options ...ReadOption) ([]uint64, error) {
	return readAllData(ch, options, DataTypeUint64, interpretUint64)
}

func (ch *Channel) ReadDataFloat32All(options ...ReadOption) ([]float32, error) {
	return readAllData(ch, options, DataTypeFloat32, interpretFloat32)
}

func (ch *Channel) ReadDataFloat64All(options ...ReadOption) ([]float64, error) {
	return readAllData(ch, options, DataTypeFloat64, interpretFloat64)
}

func (ch *Channel) ReadDataFloat128All(options ...ReadOption) ([]Float128, error) {
	return readAllData(ch, options, DataTypeFloat128, interpretFloat128)
}

func (ch *Channel) ReadDataStringAll(options ...ReadOption) ([]string, error) {
	return readAllData(ch, options, DataTypeString, interpretString)
}

func (ch *Channel) ReadDataBoolAll(options ...ReadOption) ([]bool, error) {
	return readAllData(ch, options, DataTypeBool, interpretBool)
}

func (ch *Channel) ReadDataTimeAll(options ...ReadOption) ([]time.Time, error) {
	return readAllData(ch, options, DataTypeTime, interpretTime)
}

func (ch *Channel) ReadDataComplex64All(options ...ReadOption) ([]complex64, error) {
	return readAllData(ch, options, DataTypeComplex64, interpretComplex64)
}

func (ch *Channel) ReadDataComplex128All(options ...ReadOption) ([]complex128, error) {
	return readAllData(ch, options, DataTypeComplex128, interpretComplex128)
}
