package tdms

import (
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
)

// File represents a parsed TDMS file. Use [Open] to open a file by path, or
// [New] to create a File from an [io.ReadSeeker].
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

// Group represents a group within a TDMS file, containing channels and
// properties.
type Group struct {
	Name       string
	Channels   map[string]Channel
	Properties map[string]Property

	f *File
}

// New creates a [File] from the given [io.ReadSeeker]. Set isIndex to true when
// reading a .tdms_index file. The size parameter must be the total byte length
// of the data accessible through reader.
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

// Open opens and parses the TDMS file at the given path. If the filename ends
// with ".tdms_index", it is treated as an index file. The caller must call
// [File.Close] when done.
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

// Close closes the underlying file if the File was created via [Open]. It is
// safe to call on Files created via [New] (it is a no-op in that case).
func (t *File) Close() error {
	if file, ok := t.f.(*os.File); ok && file != nil {
		return file.Close()
	}

	return nil
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
