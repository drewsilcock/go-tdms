package tdms

// This file contains the internal logic for parsing the lead in and metadata
// from the TDMS files, which is where most of the tricky code is.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
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
	leadInSize                    uint64 = 28
	daqmxFormatChangingScalerSize uint32 = 20
	daqmxDigitalLineScalerSize    uint32 = 17
)

var (
	tdmsMagicBytes      = []byte{'T', 'D', 'S', 'm'}
	tdmsIndexMagicBytes = []byte{'T', 'D', 'S', 'h'}
)

const (
	maxSupportedScalers      = 10_000
	maxSupportedDAQmxBuffers = 10_000
	maxSupportedProperties   = 100_000
)

// segment contains an individual section of the TDMS file, of which a TDMS file
// consists of one or more. Each segment has it's own lead in, optionally with
// metadata and raw data.
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

	// daqmxBufferSizes is the total size in bytes of each buffer.
	// This is equivalent to daqmxBufferWidths found in the DAQmx object raw
	// data index, where each value is multiplied by the number of values in
	// that buffer. You need to look at the scalers to see how many values are
	// in each buffer, which is why we introduce this helpful pre-calculated
	// slice.
	daqmxBufferSizes []uint64
}

type object struct {
	path string

	// If index is nil, that means there's no raw data for this object.
	index      *objectIndex
	properties Properties
}

type objectIndex struct {
	// If scaler type is none, that means this is not DAQmx data. Otherwise, it
	// is.
	daqmxScalerType daqmxScalerType
	dataType        DataType
	numValues       uint64

	// For variable-size data types, e.g. strings, this is taken from the file
	// itself. Otherwise, it is calculated from data type size and number of
	// values. This refers to the total size of this channel in bytes for a
	// single chunk.
	totalSize uint64

	// daqmxScalers maps from scaler index to scaler.
	daqmxScalers map[int]DAQmxScaler

	// daqmxBufferWidths is the size of a single sample for each DAQmx buffer in
	// bytes. These widths are the same across all objects in the segment but
	// are duplicated within each object. Multiple channels can share the same
	// buffer, so the width for a single buffer is the sum of the size of a
	// single sample from each channel. Remember, a single channel can consist
	// of multiple values interleaved together via multiple scalers.
	daqmxBufferWidths []uint32

	// Offset is the absolute offset from the beginning of the file.
	offset int64

	// Stride is the distance from one data point to the next, when the data is
	// interleaved. It is equal to the size of a single datum for all objects
	// other than the current object.
	stride int64
}

// readSegmentLeadIn reads the "lead in" data for a segment, which contains
// flags telling you how to read the rest of the segment. We need the previous
// segment because certain metadata is "carried over" from one segment to the
// next, like objects and indices.
func (t *File) readSegmentLeadIn() (*leadIn, error) {
	leadInBytes := make([]byte, leadInSize)
	if _, err := t.r.Read(leadInBytes); err != nil {
		return nil, errors.Join(ErrReadFailed, err)
	}

	magicBytes := leadInBytes[:4]
	if t.isIndex {
		if !bytes.Equal(magicBytes, tdmsIndexMagicBytes) {
			return nil, errors.Join(ErrInvalidFileFormat, errors.New("invalid TDMS index magic bytes"))
		}
	} else if !bytes.Equal(magicBytes, tdmsMagicBytes) {
		return nil, errors.Join(ErrInvalidFileFormat, errors.New("invalid TDMS magic bytes"))
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

func (t *File) readSegmentMetadata(segmentOffset int64, leadIn *leadIn, prevSegment *segment) (*metadata, error) {
	numObjects, err := readUint32(t.r, leadIn.byteOrder)
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

	obj.path, err = readString(t.r, leadIn.byteOrder)
	if err != nil {
		return nil, err
	}

	rawDataIndexHeader, err := readUint32(t.r, leadIn.byteOrder)
	if err != nil {
		return nil, err
	}

	rawDataIndexPresent := false

	switch rawDataIndexHeader {
	case rawIndexHeaderNoRawData:
		obj.index = nil
		rawDataIndexPresent = false
	case rawIndexHeaderMatchesPreviousValue:
		if prevSegment == nil {
			return nil, errors.New("raw data index matches previous value but no prior segment found")
		}

		if existingObj, ok := prevSegment.metadata.objects[obj.path]; ok {
			// We don't bother copying the index because we won't change it.
			obj.index = existingObj.index
		} else {
			return nil, errors.New("raw data index matches previous value but no prior object found")
		}

		rawDataIndexPresent = false
	case rawIndexHeaderFormatChangingScaler:
		obj.index = &objectIndex{daqmxScalerType: daqmxScalerTypeFormatChanging}
		rawDataIndexPresent = true
	case rawIndexHeaderDigitalLineScaler:
		obj.index = &objectIndex{daqmxScalerType: daqmxScalerTypeDigitalLine}
		rawDataIndexPresent = true
	default:
		// Value is the length of the raw data index. This value seems pointless
		// as the raw data index at this point is always 20 = 0x14 bytes in
		// length (including the header). I guess it's just to differentiate it
		// from the special values above, although it seems they should've then
		// used a special value to indicate "this is a normal raw data index".
		// It's probably historical.
		obj.index = &objectIndex{daqmxScalerType: daqmxScalerTypeNone}
		rawDataIndexPresent = true
	}

	if rawDataIndexPresent {
		// The normal index is always 16 bytes long so just read it all at once.
		rawDataIndexBytes := make([]byte, 16)
		if _, err := t.r.Read(rawDataIndexBytes); err != nil {
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

		if obj.index.daqmxScalerType == daqmxScalerTypeNone {
			// The total size is only present when the data size is variable,
			// e.g. is a string. I can't see any other variable size data types,
			// although I am not sure about FixedPointer and DAQmx data.
			if obj.index.dataType.Size() == 0 {
				obj.index.totalSize, err = readUint64(t.r, leadIn.byteOrder)
				if err != nil {
					return nil, errors.Join(ErrReadFailed, err)
				}
			} else {
				obj.index.totalSize = obj.index.numValues * uint64(obj.index.dataType.Size())
			}
		} else {
			numScalers, err := readUint32(t.r, leadIn.byteOrder)
			if err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			// If you've got thousands of scalers, it means your data is corrupted.
			if numScalers > maxSupportedScalers {
				return nil, fmt.Errorf("%w: unsupported number of DAQmx scalers: %d", ErrInvalidFileFormat, numScalers)
			}

			obj.index.daqmxScalers = make(map[int]DAQmxScaler, numScalers)

			// All the scalers all have the same type.
			daqmxScalerSize := daqmxFormatChangingScalerSize
			if obj.index.daqmxScalerType == daqmxScalerTypeDigitalLine {
				daqmxScalerSize = daqmxDigitalLineScalerSize
			}

			scalersBytes := make([]byte, daqmxScalerSize*numScalers)
			if _, err := t.r.Read(scalersBytes); err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			for i := range numScalers {
				// According to NI docs, TDMS file format always tores sample
				// format bitmap as uint32. According to npTDMS code, the sample
				// format bitmap is a uint32 when DAQmx scaler is a format
				// changing scaler and is a uint8 when DAQmx scaler is a digital
				// line scaler. Between the 2, I trust npTDMS more on this.
				scalerBytes := scalersBytes[i*daqmxScalerSize : (i+1)*daqmxScalerSize]

				scaler := DAQmxScaler{}
				scaler.dataType = DAQmxDataType(leadIn.byteOrder.Uint32(scalerBytes))
				scaler.rawBufferIndex = leadIn.byteOrder.Uint32(scalerBytes[4:8])
				scaler.offsetWithinStride = leadIn.byteOrder.Uint32(scalerBytes[8:12])

				if obj.index.daqmxScalerType == daqmxScalerTypeFormatChanging {
					scaler.sampleFormatBitmap = leadIn.byteOrder.Uint32(scalerBytes[12:16])
					scaler.scaleID = leadIn.byteOrder.Uint32(scalerBytes[16:])
				} else {
					scaler.sampleFormatBitmap = uint32(scalerBytes[12])
					scaler.scaleID = leadIn.byteOrder.Uint32(scalerBytes[13:])
				}

				obj.index.daqmxScalers[int(scaler.scaleID)] = scaler
			}

			numBufferWidths, err := readUint32(t.r, leadIn.byteOrder)
			if err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			if numBufferWidths > maxSupportedDAQmxBuffers {
				return nil, fmt.Errorf("%w: unsupported number of buffer widths: %d", ErrInvalidFileFormat, numBufferWidths)
			}

			obj.index.daqmxBufferWidths = make([]uint32, numBufferWidths)

			widthsBytes := make([]byte, 4*numBufferWidths)
			if _, err := t.r.Read(widthsBytes); err != nil {
				return nil, errors.Join(ErrReadFailed, err)
			}

			// The DAQmx buffer widths should be exactly the same for all
			// objects in a segment (remember, you can't mix DAQmx and non-DAQmx
			// channels in the same segment). We don't currently validate this.
			for i := range numBufferWidths {
				obj.index.daqmxBufferWidths[i] = leadIn.byteOrder.Uint32(widthsBytes[i*4:])
			}

			obj.index.totalSize = 0
			for _, scaler := range obj.index.daqmxScalers {
				obj.index.totalSize += uint64(scaler.dataType.Size()) * obj.index.numValues
			}
		}
	}

	numProps, err := readUint32(t.r, leadIn.byteOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to read number of properties: %w", err)
	}

	if numProps > maxSupportedProperties {
		return nil, fmt.Errorf("%w: unsupported number of properties: %d", ErrInvalidFileFormat, numProps)
	}

	obj.properties = make(map[string]Property, numProps)
	for range numProps {
		propName, err := readString(t.r, leadIn.byteOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to read property name: %w", err)
		}

		propDataTypeInt, err := readUint32(t.r, leadIn.byteOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to read property data type: %w", err)
		}

		propDataType := DataType(propDataTypeInt)

		value, err := readValue(propDataType, t.r, leadIn.byteOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to read property value: %w", err)
		}

		prop := Property{
			Name:  propName,
			Type:  propDataType,
			Value: value,
		}

		obj.properties[propName] = prop
	}

	return &obj, nil
}
