package tdms

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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

const leadInSize uint64 = 28

var (
	tdmsMagicBytes      []byte = []byte{'T', 'D', 'S', 'm'}
	tdmsIndexMagicBytes []byte = []byte{'T', 'D', 'S', 'h'}

	ErrUnsupportedVersion error = errors.New("unsupported version")
	ErrReadFailed         error = errors.New("failed to read data")
	ErrInvalidFileFormat  error = errors.New("invalid file format")
)

type File struct {
	Groups       []Group
	Properties   map[string]DataType
	IsIncomplete bool

	f       io.ReadSeeker
	size    int64
	isIndex bool
	err     error
}

type Group struct {
	Name       string
	Channels   []Channel
	Properties map[string]DataType

	f *File
}

type Channel struct {
	Name       string
	GroupName  string
	Properties map[string]DataType

	g        *Group
	f        *File
	dataType tdsDataType
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

type object struct {
	path         string
	rawDataIndex *rawDataIndex
	properties   map[string]DataType
}

type daqmxScaler int

const (
	daqmxScalerNone daqmxScaler = iota
	daqmxScalerFormatChaning
	daqmxScalerDigitalLine
)

type rawDataIndex struct {
	scaler daqmxScaler
}

type segment struct {
	leadIn   *leadIn
	metadata *metadata
}

func New(reader io.ReadSeeker, isIndex bool, size int64) *File {
	// Properties can be overwritten from one segment to the next, so in order
	// to know the objects and properties, we need to read the metadata for each
	// segment upfront. For ease of use, we do this here.
	f := &File{
		f:       reader,
		size:    size,
		isIndex: isIndex,
		err:     nil,
	}

	f.readMetadata()
	return f
}

func Open(fname string) (*File, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", fname, err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", fname, err)
	}

	return New(
		file,
		strings.HasSuffix(fname, ".tdms_index"),
		fileInfo.Size(),
	), nil
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

func (t *File) readSegmentMetadata(leadIn *leadIn, prevSegment *segment) (*metadata, error) {
	existingObjects := map[string]object{}
	if !leadIn.newObjectList {
		existingObjects = prevSegment.metadata.objectMap
	}

	numObjects := new(TDSUint32)
	if err := numObjects.Read(t.f, leadIn.byteOrder); err != nil {
		return nil, err
	}

	objects := make([]object, uint32(*numObjects))

	i := uint32(0)

	for i < uint32(*numObjects) {
		objectPath := new(TDSString)
		if err := objectPath.Read(t.f, leadIn.byteOrder); err != nil {
			return nil, err
		}

		// TODO: raw data index

		// TODO: num properties
		// TODO: properties

		objects[i] = object{
			path:         string(*objectPath),
			rawDataIndex: nil,
			properties:   nil,
		}

		i++
	}

	return &metadata{objects: objects}, nil
}

// readMetadata reads the metadata for each segment in the file.
func (t *File) readMetadata() error {
	var prevSegment *segment
	i := 0
	currentOffset := int64(0)

	t.f.Seek(0, io.SeekStart)

	for {
		leadIn, err := t.readSegmentLeadIn()
		if err != nil {
			return fmt.Errorf("failed to read segment %d lead in: %w", i, err)
		}

		metadata, err := t.readSegmentMetadata(leadIn, prevSegment)
		if err != nil {
			return fmt.Errorf("failed to read segment %d metadata: %w", i, err)
		}

		segment := segment{leadIn: leadIn, metadata: metadata}

		currentOffset += int64(segment.leadIn.nextSegmentOffset)

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

		prevSegment = &segment

		// If we're reading an index file, there's no data so one segment's
		// metadata leads directly into the next segment's lead in.
		if !t.isIndex {
			t.f.Seek(int64(leadIn.nextSegmentOffset), io.SeekCurrent)
		}
	}

	return nil
}
func (s *segment) Data(yield func(v []byte) bool) {
	// TODO
}
