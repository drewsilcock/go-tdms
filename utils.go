package tdms

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

func readUint32(reader io.Reader, order binary.ByteOrder) (uint32, error) {
	valueBytes := make([]byte, 4)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return order.Uint32(valueBytes), nil
}

func readUint64(reader io.Reader, order binary.ByteOrder) (uint64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return order.Uint64(valueBytes), nil
}

func readString(reader io.Reader, order binary.ByteOrder) (string, error) {
	length, err := readUint32(reader, order)
	if err != nil {
		return "", err
	}

	strBytes := make([]byte, length)
	if _, err := reader.Read(strBytes); err != nil {
		return "", errors.Join(ErrReadFailed, err)
	}

	return string(strBytes), nil
}

func parsePath(path string) (string, string, error) {
	// Each element of the path is in single quotes. Single quotes inside this
	// are escaped using two single quotes. Slashes inside single quotes don't
	// delimit the path components.

	components := make([]string, 0, 2)

	i := 0
	for {
		char := path[i]
		nextChar := byte(0)
		if i+1 < len(path) {
			nextChar = path[i+1]
		}

		if char != '/' {
			return "", "", ErrInvalidPath
		}

		if nextChar == 0 {
			// This is a root level path with no group or channel components.
			break
		}

		if nextChar != '\'' {
			return "", "", ErrInvalidPath
		}

		i += 1

		component := strings.Builder{}

		// Inner loop captures the name of the group/channel.
		for {
			char = path[i]
			nextChar = byte(0)
			if i+1 < len(path) {
				nextChar = path[i+1]
			}

			if char == '\'' {
				if nextChar == '\'' {
					// This quote is escaped, so skip forward another char.
					i += 1
				} else {
					components = append(components, component.String())
					break
				}
			}

			component.WriteByte(char)

			i += 1
		}

		i += 1
	}

	groupName := ""
	if len(components) > 0 {
		groupName = components[0]
	}

	channelName := ""
	if len(components) > 1 {
		channelName = components[1]
	}

	return groupName, channelName, nil
}
