package tdms

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
)

// This code would be much simpler if we used `binary.Read()`, but that function
// is very slow because it uses reflection.

func readInt8(reader io.Reader, order binary.ByteOrder) (int8, error) {
	valueBytes := make([]byte, 1)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return int8(valueBytes[0]), nil
}

func readInt16(reader io.Reader, order binary.ByteOrder) (int16, error) {
	valueBytes := make([]byte, 2)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return int16(order.Uint16(valueBytes)), nil
}

func readInt32(reader io.Reader, order binary.ByteOrder) (int32, error) {
	valueBytes := make([]byte, 4)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return int32(order.Uint32(valueBytes)), nil
}

func readInt64(reader io.Reader, order binary.ByteOrder) (int64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return int64(order.Uint64(valueBytes)), nil
}

func readUint8(reader io.Reader, order binary.ByteOrder) (uint8, error) {
	valueBytes := make([]byte, 1)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return valueBytes[0], nil
}

func readUint16(reader io.Reader, order binary.ByteOrder) (uint16, error) {
	valueBytes := make([]byte, 2)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return order.Uint16(valueBytes), nil
}

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

func readFloat32(reader io.Reader, order binary.ByteOrder) (float32, error) {
	valueBytes := make([]byte, 4)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return math.Float32frombits(order.Uint32(valueBytes)), nil
}

func readFloat64(reader io.Reader, order binary.ByteOrder) (float64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return math.Float64frombits(order.Uint64(valueBytes)), nil
}

func readFloat128(reader io.Reader, order binary.ByteOrder) (Float128, error) {
	valueBytes := make([]byte, 16)
	if _, err := reader.Read(valueBytes); err != nil {
		return Float128{}, errors.Join(ErrReadFailed, err)
	}

	// There no `order.Uint128()` to do this for us, so just reverse the slice.
	// Probably not as fast as the bit shifting method from binary.LittleEndian,
	// but hey. We store the value as little endian so it's standardised and we
	// don't need to know the byte order when we convert it to another type.
	if order == binary.BigEndian {
		slices.Reverse(valueBytes)
	}

	return Float128(valueBytes), nil
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

func readBool(reader io.Reader, order binary.ByteOrder) (bool, error) {
	valueBytes := make([]byte, 1)
	if _, err := reader.Read(valueBytes); err != nil {
		return false, errors.Join(ErrReadFailed, err)
	}

	return valueBytes[0] != 0, nil
}

func readTime(reader io.Reader, order binary.ByteOrder) (Time, error) {
	valueBytes := make([]byte, 16)
	if _, err := reader.Read(valueBytes); err != nil {
		return Time{}, errors.Join(ErrReadFailed, err)
	}

	return Time{
		Timestamp: int64(order.Uint64(valueBytes)),
		Remainder: order.Uint64(valueBytes[8:]),
	}, nil
}

func readComplex64(reader io.Reader, order binary.ByteOrder) (complex64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0 + 0i, errors.Join(ErrReadFailed, err)
	}

	realValue := math.Float32frombits(order.Uint32(valueBytes))
	imagValue := math.Float32frombits(order.Uint32(valueBytes[4:]))

	return complex(realValue, imagValue), nil
}

func readComplex128(reader io.Reader, order binary.ByteOrder) (complex128, error) {
	valueBytes := make([]byte, 16)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0 + 0i, errors.Join(ErrReadFailed, err)
	}

	realValue := math.Float64frombits(order.Uint64(valueBytes))
	imagValue := math.Float64frombits(order.Uint64(valueBytes[8:]))

	return complex(realValue, imagValue), nil
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

		// Skip over the / and the '
		i += 2

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

		if i >= len(path) {
			break
		}
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
