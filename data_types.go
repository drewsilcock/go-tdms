package tdms

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"
)

// DataType represents the type of data stored in a TDMS channel or property.
type DataType uint32

const (
	// DataTypeVoid represents an empty or void data type.
	DataTypeVoid DataType = iota

	// DataTypeInt8 represents an 8-bit signed integer.
	DataTypeInt8

	// DataTypeInt16 represents a 16-bit signed integer.
	DataTypeInt16

	// DataTypeInt32 represents a 32-bit signed integer.
	DataTypeInt32

	// DataTypeInt64 represents a 64-bit signed integer.
	DataTypeInt64

	// DataTypeUint8 represents an 8-bit unsigned integer.
	DataTypeUint8

	// DataTypeUint16 represents a 16-bit unsigned integer.
	DataTypeUint16

	// DataTypeUint32 represents a 32-bit unsigned integer.
	DataTypeUint32

	// DataTypeUint64 represents a 64-bit unsigned integer.
	DataTypeUint64

	// DataTypeFloat32 represents a 32-bit floating point number.
	DataTypeFloat32

	// DataTypeFloat64 represents a 64-bit floating point number.
	DataTypeFloat64

	// DataTypeFloat128 represents a 128-bit extended precision floating point number.
	DataTypeFloat128

	// DataTypeFloat32WithUnit represents a 32-bit floating point number with an associated unit string property.
	DataTypeFloat32WithUnit DataType = 0x19

	// DataTypeFloat64WithUnit represents a 64-bit floating point number with an associated unit string property.
	DataTypeFloat64WithUnit DataType = 0x1A

	// DataTypeFloat128WithUnit represents a 128-bit floating point number with an associated unit string property.
	DataTypeFloat128WithUnit DataType = 0x1B

	// DataTypeString represents a variable-length UTF-8 string.
	DataTypeString DataType = 0x20

	// DataTypeBool represents a boolean value.
	DataTypeBool DataType = 0x21

	// DataTypeTimestamp represents a timestamp with extended precision.
	DataTypeTimestamp DataType = 0x44

	// DataTypeFixedPoint represents a fixed-point number (not currently supported).
	DataTypeFixedPoint DataType = 0x4F

	// DataTypeComplex64 represents a complex number with 32-bit real and imaginary parts.
	DataTypeComplex64 DataType = 0x08000c

	// DataTypeComplex128 represents a complex number with 64-bit real and imaginary parts.
	DataTypeComplex128 DataType = 0x10000d

	// DataTypeDAQmxRawData represents raw DAQmx data.
	DataTypeDAQmxRawData DataType = 0xFFFFFFFF
)

const fractionsPerNs uint64 = 18_446_744_073 // 2 ** 64 / 10 ** 9

var tdmsEpoch int64 = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

// Anything with exponent filled with 1s and non-zero mantissa is a NaN – this
// is the one we choose to use.
var quadNaN = []byte{
	0x7f, 0xff, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
}

// Size returns the size in bytes of a single value of this data type.
// Returns 0 for variable-length types like strings.
func (dt DataType) Size() int {
	switch dt {
	case DataTypeVoid, DataTypeString:
		return 0 // Strings are variable length
	case DataTypeInt8, DataTypeUint8, DataTypeBool:
		return 1
	case DataTypeInt16, DataTypeUint16:
		return 2
	case DataTypeInt32, DataTypeUint32, DataTypeFloat32:
		return 4
	case DataTypeInt64, DataTypeUint64, DataTypeFloat64, DataTypeComplex64:
		return 8
	case DataTypeFloat128, DataTypeComplex128, DataTypeTimestamp:
		return 16
	default:
		return 0
	}
}

// String implements the [fmt.Stringer] interface, returning the human-readable
// name of the data type.
func (dt DataType) String() string {
	return dt.Name()
}

// Name returns the human-readable name of the data type.
func (dt DataType) Name() string {
	switch dt {
	case DataTypeVoid:
		return "Void"
	case DataTypeInt8:
		return "Int8"
	case DataTypeInt16:
		return "Int16"
	case DataTypeInt32:
		return "Int32"
	case DataTypeInt64:
		return "Int64"
	case DataTypeUint8:
		return "Uint8"
	case DataTypeUint16:
		return "Uint16"
	case DataTypeUint32:
		return "Uint32"
	case DataTypeUint64:
		return "Uint64"
	case DataTypeFloat32:
		return "Float32"
	case DataTypeFloat64:
		return "Float64"
	case DataTypeFloat128, DataTypeFloat128WithUnit:
		return "Float128"
	case DataTypeString:
		return "String"
	case DataTypeBool:
		return "Bool"
	case DataTypeTimestamp:
		return "Time"
	case DataTypeComplex64:
		return "ComplexFloat64"
	case DataTypeComplex128:
		return "ComplexFloat128"
	case DataTypeFixedPoint:
		return "FixedPoint"
	case DataTypeDAQmxRawData:
		return "DAQmxRawData"
	default:
		return fmt.Sprintf("Unknown(0x%X)", uint32(dt))
	}
}

func readValue(typeCode DataType, reader io.Reader, byteOrder binary.ByteOrder) (any, error) {
	switch typeCode {
	case DataTypeVoid:
		return nil, nil
	case DataTypeInt8:
		return readInt8(reader, byteOrder)
	case DataTypeInt16:
		return readInt16(reader, byteOrder)
	case DataTypeInt32:
		return readInt32(reader, byteOrder)
	case DataTypeInt64:
		return readInt64(reader, byteOrder)
	case DataTypeUint8:
		return readUint8(reader, byteOrder)
	case DataTypeUint16:
		return readUint16(reader, byteOrder)
	case DataTypeUint32:
		return readUint32(reader, byteOrder)
	case DataTypeUint64:
		return readUint64(reader, byteOrder)
	// The "with unit" data types are exactly the same, but just tell readers to
	// exact the unit to be in a property called "unit_string".
	case DataTypeFloat32, DataTypeFloat32WithUnit:
		return readFloat32(reader, byteOrder)
	case DataTypeFloat64, DataTypeFloat64WithUnit:
		return readFloat64(reader, byteOrder)
	case DataTypeFloat128, DataTypeFloat128WithUnit:
		return readFloat128(reader, byteOrder)
	case DataTypeString:
		return readString(reader, byteOrder)
	case DataTypeBool:
		return readBool(reader, byteOrder)
	case DataTypeTimestamp:
		return readTime(reader, byteOrder)
	case DataTypeComplex64:
		return readComplex64(reader, byteOrder)
	case DataTypeComplex128:
		return readComplex128(reader, byteOrder)
	default:
		return nil, ErrUnsupportedType
	}

	// The NI documentation provides nothing on how fixed points are stored.
	// There is a page for how they are stored in memory while using LabVIEW,
	// but not how it is stored on disk. Without an example or additional
	// documentation, it's not possible to implement this. It's also not
	// possible to know how large the data points are, which means you can't
	// know how far to skip even if you want to ignore the fixed point channel.
	//
	// This means that the presence of any fixed point data renders a file
	// unreadable. If you have more information or an actual TDMS file with a
	// fixed point data channel in it, please contact the author of this
	// repository so that this can be implemented.
	//
	// See:
	// https://www.ni.com/docs/en-US/bundle/labview/page/numeric-data.html
	// https://www.ni.com/docs/en-US/bundle/labview/page/numeric-data-types-table.html
	// https://www.ni.com/docs/en-US/bundle/labview/page/labview-manager-data-types.html#d96127e328

	// It's also not clear what DAQmx data type actually means – is it just an
	// indicator that the data is a vector of other data types?
}

// Float128 represents a 128-bit extended precision floating point number.
// When represented in memory, this type is always little endian. To get a
// usable value, see [Float128.AsFloat64] and [Float128.AsBigFloat], depending
// on whether you need the full precision or not.
//
// The bytes underlying this type correspond to a little-endian IEEE quad-precision
// floating point number, with 1-bit sign, 15-bit exponent and 112-bit mantissa.
type Float128 [16]byte

// String implements the [fmt.Stringer] interface, returning the string representation
// of the [Float128] value via [big.Float].
func (f Float128) String() string {
	return f.AsBigFloat().String()
}

func (f Float128) IsNaN() bool {
	exponent := int(f[0]&0x7F)<<8 | int(f[1])
	return exponent == 0x7fff && !isZeroMantissa(f[2:16])
}

// AsFloat64 converts the 128-bit extended precision float to a primitive float64.
// This loses a significant amount of precision. To avoid losing any precision
// at the cost of usability, see [Float128.AsBigFloat].
func (f Float128) AsFloat64() float64 {
	// Big float doesn't handle NaN so we have to handle this separately.
	if f.IsNaN() {
		return math.NaN()
	}

	result, _ := f.AsBigFloat().Float64()
	return result
}

// AsBigFloat converts the 128-bit extended precision float to a big.Float.
// This is the most precise representation of the value, meaning no precision is
// lost in the conversion to big.Float.
//
// If you do not require the full precision of the original 128-bit floating
// point number, you can use the [Float128.AsFloat64] method to convert the
// float to a 64-bit number, losing precision at the benefit of ease of use.
func (f Float128) AsBigFloat() *big.Float {
	// Extract sign bit (bit 127)
	sign := (f[0] >> 7) & 1

	// Extract exponent (bits 126-112, 15 bits total)
	exponent := uint16(f[0]&0x7F) << 8
	exponent |= uint16(f[1])

	// Extract mantissa (bits 111-0, 112 bits)
	mantissaBits := f[2:16]

	// Quad precision has 113 bits of precision according to IEEE (remember
	// implicit leading 1).
	result := new(big.Float).SetPrec(113)

	// Handle special case of nan/inf
	if exponent == 0x7FFF {
		if isZeroMantissa(mantissaBits) {
			return result.SetInf(sign == 1)
		} else {
			// big.Float can't handle NaN values.
			return nil
		}
	}

	shiftAmount := new(big.Int).Lsh(big.NewInt(1), 112)

	if exponent == 0 {
		// Subnormal or zero
		if isZeroMantissa(mantissaBits) {
			return result.SetInt64(0)
		}

		// Subnormal number: exponent is -16382, implicit leading bit is 0
		result.SetFloat64(0)
		mantissaValue := mantissaToBigInt(mantissaBits)
		mantissaFloat := new(big.Float).SetInt(mantissaValue)
		mantissaFloat.Quo(mantissaFloat, new(big.Float).SetInt(shiftAmount))

		power := new(big.Float).SetMantExp(big.NewFloat(1), -16382)
		result.Mul(mantissaFloat, power)

		if sign == 1 {
			result.Neg(result)
		}

		return result
	}

	// Normal number: implicit leading bit is 1
	exponentValue := int(exponent) - 16383
	mantissaValue := mantissaToBigInt(mantissaBits)

	// Combine: (1.mantissa) * 2^exponent
	mantissaFloat := new(big.Float).SetInt(mantissaValue)
	mantissaFloat.Quo(mantissaFloat, new(big.Float).SetInt(shiftAmount))
	mantissaFloat.Add(mantissaFloat, big.NewFloat(1))

	// Apply exponent – you could directly apply SetMantExp() to result here,
	// but it would override any other properties set on result such as the
	// precision from the mantissaFloat.
	power := new(big.Float).SetMantExp(big.NewFloat(1), exponentValue)
	result.Mul(mantissaFloat, power)

	// Apply sign
	if sign == 1 {
		result.Neg(result)
	}

	return result
}

// SetBigFloat sets the Float128 value from a [big.Float].
// The conversion handles special cases like infinity and zero, and encodes
// the value into IEEE 754 quad-precision (binary128) format.
func (f *Float128) SetBigFloat(bf *big.Float) *Float128 {
	// Clear all bytes first
	for i := range f {
		f[i] = 0
	}

	if bf == nil {
		return f
	}

	if bf.IsInf() {
		// Set exponent to all 1s (0x7FFF)
		f[0] = 0x7F
		f[1] = 0xFF

		// Set sign bit if negative infinity
		if bf.Signbit() {
			f[0] |= 0x80
		}
		return f
	}

	// Big floats cannot be NaN so we don't need to handle that case.

	// Handle zero
	if bf.Sign() == 0 {
		if bf.Signbit() {
			f[0] |= 0x80
		}
		return f
	}

	// Extract sign
	sign := bf.Signbit()

	// Get the mantissa and exponent from big.Float
	// bf = mantissa * 2^exponent where 0.5 <= |mantissa| < 1.0
	mantissa := new(big.Float).SetPrec(113).Copy(bf)
	if sign {
		mantissa.Neg(mantissa)
	}

	// Get the actual exponent by using MantExp
	// This returns mantissa in range [0.5, 1) and the exponent
	exponent := mantissa.MantExp(mantissa)

	// IEEE 754 quad precision format:
	// - Normalized: 1.fraction * 2^(exponent - bias) where bias = 16383
	// - Mantissa returned from MantExp is in [0.5, 1), we need [1, 2)
	// So multiply by 2 and adjust exponent
	mantissa.Mul(mantissa, big.NewFloat(2))
	exponent--

	// Check for overflow/underflow
	// Exponent range for normalized numbers: -16382 to 16383
	// With bias (16383): 1 to 32766 (0x1 to 0x7FFE)
	const maxExponent = 16383
	const minExponent = -16382

	if exponent > maxExponent {
		// Overflow to infinity
		f[0] = 0x7F
		f[1] = 0xFF
		if sign {
			f[0] |= 0x80
		}
		return f
	}

	// Handle subnormal numbers (exponent too small for normalized form)
	var biasedExponent uint16
	var fraction *big.Float

	if exponent < minExponent {
		// Subnormal number: exponent = 0, no implicit leading 1
		biasedExponent = 0

		// Calculate the actual fraction for subnormal
		// For subnormal: value = fraction * 2^(-16382)
		// We have: mantissa * 2^exponent
		// So: fraction = mantissa * 2^(exponent - (-16382)) = mantissa * 2^(exponent + 16382)
		shift := exponent + 16382
		fraction = new(big.Float).SetPrec(113).Copy(mantissa)
		if shift < 0 {
			// Number too small, underflow to zero
			return f
		}
		// Apply the shift using SetMantExp
		scaler := new(big.Float).SetMantExp(big.NewFloat(1), shift)
		fraction.Mul(fraction, scaler)
	} else {
		// Normal number
		biasedExponent = uint16(exponent + 16383)

		// Extract fractional part (mantissa is in [1, 2), we want the fraction after 1)
		fraction = new(big.Float).SetPrec(113).Sub(mantissa, big.NewFloat(1))
	}

	// Convert fraction to 112-bit integer
	// Multiply by 2^112 to get the integer representation
	scaler := new(big.Int).Lsh(big.NewInt(1), 112)
	fraction.Mul(fraction, new(big.Float).SetInt(scaler))

	// Convert to integer (with rounding)
	fractionInt, _ := fraction.Int(nil)

	// Handle potential rounding overflow
	// If fraction rounded up to 2^112 or beyond, we need to increment exponent
	maxFraction := new(big.Int).Lsh(big.NewInt(1), 112)
	if fractionInt.Cmp(maxFraction) >= 0 {
		if biasedExponent == 0 {
			// Subnormal rounded up to normal
			biasedExponent = 1
			fractionInt.SetInt64(0)
		} else if biasedExponent < 0x7FFE {
			// Normal number, increment exponent
			biasedExponent++
			fractionInt.SetInt64(0)
		} else {
			// Overflow to infinity
			f[0] = 0x7F
			f[1] = 0xFF
			if sign {
				f[0] |= 0x80
			}
			return f
		}
	}

	// Encode the 112-bit mantissa into bytes 2-15
	// Looking at mantissaToBigInt, it processes bytes in order: f[2], f[3], ..., f[15]
	// shifting left each time. This means f[2] is the MSB and f[15] is the LSB.
	// So we store the mantissa in big-endian format starting at f[2].
	fractionBytes := fractionInt.Bytes() // Big-endian from math/big

	// Pad with leading zeros if needed
	// We need exactly 14 bytes for the 112-bit mantissa
	offset := 16 - len(fractionBytes)
	if offset < 2 {
		// Mantissa too large - take only the most significant 14 bytes
		offset = 2
		fractionBytes = fractionBytes[len(fractionBytes)-14:]
	}

	// Copy mantissa bytes to f[2:16]
	for i := 0; i < len(fractionBytes); i++ {
		f[offset+i] = fractionBytes[i]
	}

	// Encode exponent into bits 112-126 (spans bytes 0-1)
	// Byte 0: sign bit (bit 7) + upper 7 bits of exponent (bits 6-0)
	// Byte 1: lower 8 bits of exponent
	f[0] = byte((biasedExponent >> 8) & 0x7F)
	f[1] = byte(biasedExponent & 0xFF)

	// Set sign bit (bit 127 = bit 7 of byte 0)
	if sign {
		f[0] |= 0x80
	}

	return f
}

func (f *Float128) SetFloat64(value float64) *Float128 {
	// big.Float cannot handle NaN, so we have to handle it ourselves.
	if math.IsNaN(value) {
		*f = Float128(quadNaN)
		return f
	}

	return f.SetBigFloat(big.NewFloat(value))
}

func NewFloat128(value float64) Float128 {
	return *new(Float128).SetFloat64(value)
}

func isZeroMantissa(mantissaBits []byte) bool {
	for _, b := range mantissaBits {
		if b != 0 {
			return false
		}
	}
	return true
}

func mantissaToBigInt(mantissaBits []byte) *big.Int {
	result := new(big.Int)
	for _, b := range mantissaBits {
		result.Lsh(result, 8)
		result.Or(result, new(big.Int).SetInt64(int64(b)))
	}
	return result
}

// Timestamp is the TDMS representation of timestamps.
//
// TDMS timestamps have significantly more precision than a standard time.Time
// value. Be aware that when converting to [time.Time] using [Timestamp.AsTime],
// precision is lost. For most purposes, this is acceptable as the level of
// precision in the TDMS format is extreme (the lowest representable value is
// roughly half an attosecond, compared to time.Time which can store no less
// than one nanosecond).
//
// TDMS stores timestamps as a combination of int64 number of seconds since TDMS
// epoch (which is 1st January 1904 at midnight) and uint64 number representing
// the fractional remainder, where the actual fractional number of seconds is
// retrieved by dividing by 2^64. There is no timezone support.
//
// For details, see:
// https://www.ni.com/en/support/documentation/supplemental/08/labview-timestamp-overview.html
type Timestamp struct {
	// Timestamp is the number of seconds since the TDMS epoch (January 1, 1904
	// 00:00:00 UTC), excluding any leap seconds.
	Timestamp int64

	// Remainder is the fractional part of the timestamp, where the actual fraction
	// is Remainder / 2^64.
	Remainder uint64
}

// AsTime converts the TDMS timestamp to a [time.Time] value. This removes much
// of the precision in the TDMS timestamp by converting from uint64 remainder
// value (which represents 2^-64ths of a second, approximately 0.05 attoseconds)
// to nanoseconds. The TDMS format retains approximately 1.8 × 10^10 times more
// information than [time.Time]. This precision loss is not relevant for most
// purposes, but important to keep in mind for high-precision applications.
func (t Timestamp) AsTime() time.Time {
	return time.Unix(t.Timestamp+tdmsEpoch, int64(t.Remainder/fractionsPerNs)).UTC()
}

// String implements the [fmt.Stringer] interface, returning the string
// representation of the timestamp as a time.Time value.
func (t Timestamp) String() string {
	return t.AsTime().String()
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{
		Timestamp: t.Unix() - tdmsEpoch,
		Remainder: uint64(t.Nanosecond()) * fractionsPerNs,
	}
}
