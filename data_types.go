package tdms

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"time"
)

type DataType uint32

const (
	DataTypeVoid DataType = iota
	DataTypeInt8
	DataTypeInt16
	DataTypeInt32
	DataTypeInt64
	DataTypeUint8
	DataTypeUint16
	DataTypeUint32
	DataTypeUint64
	DataTypeFloat32
	DataTypeFloat64
	DataTypeFloat128
	DataTypeFloat32WithUnit  DataType = 0x19
	DataTypeFloat64WithUnit  DataType = 0x1A
	DataTypeFloat128WithUnit DataType = 0x1B
	DataTypeString           DataType = 0x20
	DataTypeBool             DataType = 0x21
	DataTypeTimestamp        DataType = 0x44
	DataTypeFixedPoint       DataType = 0x4F
	DataTypeComplex64        DataType = 0x08000c
	DataTypeComplex128       DataType = 0x10000d
	DataTypeDAQmxRawData     DataType = 0xFFFFFFFF
)

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

// When represented in memory, this type is always little endian. To get a
// usable value, see `Float64()` and `BigFloat()`, depending on whether you need
// the full precision or not.
type Float128 [16]byte

// Float64 converts the 128-bit extended precision float to a primitive float64.
// This loses a significant amount of precision. To avoid losing any precision
// at the cost of usability, see `BigFloat()`.
func (f Float128) Float64() float64 {
	result, _ := f.BigFloat().Float64()
	return result
}

func (f Float128) BigFloat() *big.Float {
	// Extract sign bit (bit 127)
	sign := (f[0] >> 7) & 1

	// Extract exponent (bits 126-112, 15 bits total)
	exponent := uint16(f[0]&0x7F) << 8
	exponent |= uint16(f[1])

	// Extract mantissa (bits 111-0, 112 bits)
	mantissaBits := make([]byte, 14)
	copy(mantissaBits, f[2:16])

	// Quad precision has 113 bits of precision according to IEEE
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
// TDMS timestamps have significantly more precision than a standard time.Timestamp
// value. Be aware that when converting to time.Timestamp using `AsTime()`, precision
// is lost. For most purposes, this is acceptable as the level of precision in
// the TDMS format is insane (the lowest representable value is roughly half an
// attosecond, compared to time.Timestamp which can store no less than one
// nanosecond).
//
// TDMS stores timestamps as a combination of i64 n# seconds since TDMS epoch
// which is 1st January 1904 at midnight and u64 number representing number
// fractional remainder, where the actual fractional n# seconds is retrieved by
// dividing by 2^-64. There is no timezone support.
//
// For details, see:
// https://www.ni.com/en/support/documentation/supplemental/08/labview-timestamp-overview.html
type Timestamp struct {
	Timestamp int64
	Remainder uint64
}

// AsTime removes much of the precision in the TDMS timestamp itself by converting
// from u64 remainder value (which is n# of 2^-64ths of a second =~ 0.05
// attoseconds) to nanoseconds. Thus, the TDMS format retains approximately 1.8 ×
// 10^10 times more information than time.AsTime. This is not relevant for most
// purposes, but important to keep in mind.
func (t *Timestamp) AsTime() time.Time {
	// I'm not sure whether this big.Int stuff is necessary as opposed to doing
	// `float64(posFractions) * math.Pow(2, -64) * 1e9`. I need to experiment
	// with some large values to determine.
	ns := new(big.Int).SetUint64(t.Remainder)
	ns.Mul(ns, big.NewInt(1e9))
	ns.Rsh(ns, 64)
	return time.Unix(t.Timestamp, ns.Int64())
}
