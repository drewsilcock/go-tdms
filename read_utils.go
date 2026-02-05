package tdms

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
	"time"
)

// This code would be much simpler if we used `binary.Read()`, but that function
// is very slow because it uses reflection.

// interpretBytes does not do anything to handle the byte order and just returns
// the same array that you put in, copied into a new array.
func interpretBytes(bytes []byte, order binary.ByteOrder) []byte {
	ret := make([]byte, len(bytes))
	copy(ret, bytes)
	return ret
}

func readInt8(reader io.Reader, order binary.ByteOrder) (int8, error) {
	valueBytes := make([]byte, 1)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretInt8(valueBytes, order), nil
}

func readInt16(reader io.Reader, order binary.ByteOrder) (int16, error) {
	valueBytes := make([]byte, 2)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretInt16(valueBytes, order), nil
}

func readInt32(reader io.Reader, order binary.ByteOrder) (int32, error) {
	valueBytes := make([]byte, 4)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretInt32(valueBytes, order), nil
}

func readInt64(reader io.Reader, order binary.ByteOrder) (int64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretInt64(valueBytes, order), nil
}

func readUint8(reader io.Reader, order binary.ByteOrder) (uint8, error) {
	valueBytes := make([]byte, 1)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretUint8(valueBytes, order), nil
}

func readUint16(reader io.Reader, order binary.ByteOrder) (uint16, error) {
	valueBytes := make([]byte, 2)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretUint16(valueBytes, order), nil
}

func readUint32(reader io.Reader, order binary.ByteOrder) (uint32, error) {
	valueBytes := make([]byte, 4)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretUint32(valueBytes, order), nil
}

func readUint64(reader io.Reader, order binary.ByteOrder) (uint64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretUint64(valueBytes, order), nil
}

func readFloat32(reader io.Reader, order binary.ByteOrder) (float32, error) {
	valueBytes := make([]byte, 4)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretFloat32(valueBytes, order), nil
}

func readFloat64(reader io.Reader, order binary.ByteOrder) (float64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0, errors.Join(ErrReadFailed, err)
	}

	return interpretFloat64(valueBytes, order), nil
}

func readFloat128(reader io.Reader, order binary.ByteOrder) (Float128, error) {
	valueBytes := make([]byte, 16)
	if _, err := reader.Read(valueBytes); err != nil {
		return Float128{}, errors.Join(ErrReadFailed, err)
	}

	return interpretFloat128(valueBytes, order), nil
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

	return interpretString(strBytes, order), nil
}

func readBool(reader io.Reader, order binary.ByteOrder) (bool, error) {
	valueBytes := make([]byte, 1)
	if _, err := reader.Read(valueBytes); err != nil {
		return false, errors.Join(ErrReadFailed, err)
	}

	return interpretBool(valueBytes, order), nil
}

func readTime(reader io.Reader, order binary.ByteOrder) (time.Time, error) {
	valueBytes := make([]byte, 16)
	if _, err := reader.Read(valueBytes); err != nil {
		return time.Time{}, errors.Join(ErrReadFailed, err)
	}

	return interpretTime(valueBytes, order), nil
}

func readComplex64(reader io.Reader, order binary.ByteOrder) (complex64, error) {
	valueBytes := make([]byte, 8)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0 + 0i, errors.Join(ErrReadFailed, err)
	}

	return interpretComplex64(valueBytes, order), nil
}

func readComplex128(reader io.Reader, order binary.ByteOrder) (complex128, error) {
	valueBytes := make([]byte, 16)
	if _, err := reader.Read(valueBytes); err != nil {
		return 0 + 0i, errors.Join(ErrReadFailed, err)
	}

	return interpretComplex128(valueBytes, order), nil
}

// Interpret functions - convert byte slices to their respective types

func interpretVoid(bytes []byte, order binary.ByteOrder) struct{} {
	return struct{}{}
}

func interpretInt8(bytes []byte, order binary.ByteOrder) int8 {
	return int8(bytes[0])
}

func interpretInt16(bytes []byte, order binary.ByteOrder) int16 {
	return int16(order.Uint16(bytes))
}

func interpretInt32(bytes []byte, order binary.ByteOrder) int32 {
	return int32(order.Uint32(bytes))
}

func interpretInt64(bytes []byte, order binary.ByteOrder) int64 {
	return int64(order.Uint64(bytes))
}

func interpretUint8(bytes []byte, order binary.ByteOrder) uint8 {
	return bytes[0]
}

func interpretUint16(bytes []byte, order binary.ByteOrder) uint16 {
	return order.Uint16(bytes)
}

func interpretUint32(bytes []byte, order binary.ByteOrder) uint32 {
	return order.Uint32(bytes)
}

func interpretUint64(bytes []byte, order binary.ByteOrder) uint64 {
	return order.Uint64(bytes)
}

func interpretFloat32(bytes []byte, order binary.ByteOrder) float32 {
	return math.Float32frombits(order.Uint32(bytes))
}

func interpretFloat64(bytes []byte, order binary.ByteOrder) float64 {
	return math.Float64frombits(order.Uint64(bytes))
}

func interpretFloat128(bytes []byte, order binary.ByteOrder) Float128 {
	// There no `order.Uint128()` to do this for us, so just reverse the slice.
	// Probably not as fast as the bit shifting method from binary.LittleEndian,
	// but hey. We store the value as little endian so it's standardised and we
	// don't need to know the byte order when we convert it to another type.
	if order == binary.BigEndian {
		slices.Reverse(bytes)
	}

	return Float128(bytes)
}

func interpretString(bytes []byte, order binary.ByteOrder) string {
	// This relies on you having already ascertained the length, which is stored
	// in the file either at the start of the data point or the start of the
	// chunk.
	return string(bytes)
}

func interpretBool(bytes []byte, order binary.ByteOrder) bool {
	return bytes[0] != 0
}

func interpretTime(bytes []byte, order binary.ByteOrder) time.Time {
	tdsTime := Time{
		Timestamp: int64(order.Uint64(bytes)),
		Remainder: order.Uint64(bytes[8:]),
	}
	return tdsTime.Time()
}

func interpretComplex64(bytes []byte, order binary.ByteOrder) complex64 {
	realValue := math.Float32frombits(order.Uint32(bytes))
	imagValue := math.Float32frombits(order.Uint32(bytes[4:]))

	return complex(realValue, imagValue)
}

func interpretComplex128(bytes []byte, order binary.ByteOrder) complex128 {
	realValue := math.Float64frombits(order.Uint64(bytes))
	imagValue := math.Float64frombits(order.Uint64(bytes[8:]))

	return complex(realValue, imagValue)
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
