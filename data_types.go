package tdms

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"slices"
	"time"
)

type tdsDataType uint32

const (
	tdsTypeVoid tdsDataType = iota
	tdsTypeInt8
	tdsTypeInt16
	tdsTypeInt32
	tdsTypeInt64
	tdsTypeUint8
	tdsTypeUint16
	tdsTypeUint32
	tdsTypeUint64
	tdsTypeFloat32
	tdsTypeFloat64
	tdsTypeFloat128
	tdsTypeFloat32WithUnit  tdsDataType = 0x19
	tdsTypeFloat64WithUnit  tdsDataType = 0x1A
	tdsTypeFloat128WithUnit tdsDataType = 0x1B
	tdsTypeString           tdsDataType = 0x20
	tdsTypeBoolean          tdsDataType = 0x21
	tdsTypeTime             tdsDataType = 0x44
	tdsTypeFixedPoint       tdsDataType = 0x4F
	tdsTypeComplex64        tdsDataType = 0x08000c
	tdsTypeComplex128       tdsDataType = 0x10000d
	tdsTypeDAQmxRawData     tdsDataType = 0xFFFFFFFF
)

func (dt tdsDataType) Size() int {
	switch dt {
	case tdsTypeVoid, tdsTypeString:
		return 0 // Strings are variable length
	case tdsTypeInt8, tdsTypeUint8, tdsTypeBoolean:
		return 1
	case tdsTypeInt16, tdsTypeUint16:
		return 2
	case tdsTypeInt32, tdsTypeUint32, tdsTypeFloat32:
		return 4
	case tdsTypeInt64, tdsTypeUint64, tdsTypeFloat64, tdsTypeComplex64:
		return 8
	case tdsTypeFloat128, tdsTypeComplex128, tdsTypeTime:
		return 16
	default:
		return 0
	}
}

func (dt tdsDataType) Name() string {
	switch dt {
	case tdsTypeVoid:
		return "Void"
	case tdsTypeInt8:
		return "Int8"
	case tdsTypeInt16:
		return "Int16"
	case tdsTypeInt32:
		return "Int32"
	case tdsTypeInt64:
		return "Int64"
	case tdsTypeUint8:
		return "Uint8"
	case tdsTypeUint16:
		return "Uint16"
	case tdsTypeUint32:
		return "Uint32"
	case tdsTypeUint64:
		return "Uint64"
	case tdsTypeFloat32:
		return "Float32"
	case tdsTypeFloat64:
		return "Float64"
	case tdsTypeFloat128, tdsTypeFloat128WithUnit:
		return "Float128"
	case tdsTypeString:
		return "String"
	case tdsTypeBoolean:
		return "Boolean"
	case tdsTypeTime:
		return "Time"
	case tdsTypeComplex64:
		return "ComplexFloat64"
	case tdsTypeComplex128:
		return "ComplexFloat128"
	case tdsTypeFixedPoint:
		return "FixedPoint"
	case tdsTypeDAQmxRawData:
		return "DAQmxRawData"
	default:
		return fmt.Sprintf("Unknown(0x%X)", uint32(dt))
	}
}

// This is the TDMS epoch as a unix timestamp. To convert from a TDMS timestamp
// to a unix timestamp, you can simply do `tdmsTimestamp + tdmsEpoch`.
const tdmsEpoch int64 = -2_082_844_800

func ptr[T any](value T) *T { return &value }

func NewDataType(typeCode tdsDataType) (DataType, error) {
	switch typeCode {
	case tdsTypeVoid:
		return &TDSVoid{}, nil
	case tdsTypeI8:
		return ptr(TDSInt8(0)), nil
	case tdsTypeI16:
		return ptr(TDSInt16(0)), nil
	case tdsTypeI32:
		return ptr(TDSInt32(0)), nil
	case tdsTypeI64:
		return ptr(TDSInt64(0)), nil
	case tdsTypeU8:
		return ptr(TDSUint8(0)), nil
	case tdsTypeU16:
		return ptr(TDSUint16(0)), nil
	case tdsTypeU32:
		return ptr(TDSUint32(0)), nil
	case tdsTypeU64:
		return ptr(TDSUint64(0)), nil
	case tdsTypeSingleFloat:
		return ptr(TDSFloat32(0)), nil
	case tdsTypeDoubleFloat:
		return ptr(TDSFloat64(0)), nil
	case tdsTypeExtendedFloat:
		return &TDSFloat128{}, nil
	case tdsTypeSingleFloatWithUnit:
		return ptr(TDSFloat32WithUnit(0)), nil
	case tdsTypeDoubleFloatWithUnit:
		return ptr(TDSFloat64WithUnit(0)), nil
	case tdsTypeExtendedFloatWithUnit:
		return &TDSFloat128WithUnit{}, nil
	case tdsTypeString:
		return ptr(TDSString("")), nil
	case tdsTypeBoolean:
		return ptr(TDSBool(false)), nil
	case tdsTypeTimeStamp:
		return &TDSTime{}, nil
	case tdsTypeFixedPoint:
		return &TDSFixedPoint{}, nil
	case tdsTypeComplexSingleFloat:
		return ptr(TDSComplexFloat32(0 + 0i)), nil
	case tdsTypeComplexDoubleFloat:
		return ptr(TDSComplexFloat64(0 + 0i)), nil
	case tdsTypeDAQmxRawData:
		return &TDSDAQmxRawData{}, nil
	default:
		return nil, fmt.Errorf("unknown type code: %d", typeCode)
	}
}

type DataType interface {
	// The size of the data type in bytes. Value of `-1` means the size is variable.
	Size() int

	// Read data from reader into variable, using input byte order. Assumes the
	// input reader is positioned at the start of the data type.
	Read(reader io.Reader, byteOrder binary.ByteOrder) error
}

type TDSVoid struct{}

func (t TDSVoid) Size() int {
	return 0
}

func (t TDSVoid) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	return nil
}

type TDSInt8 int8

func (t TDSInt8) Size() int {
	return 1
}

func (t *TDSInt8) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	// Byte order doesn't matter here because it's only 1 byte long.
	*t = TDSInt8(int8(valBytes[0]))
	return nil
}

type TDSInt16 int16

func (t TDSInt16) Size() int {
	return 2
}

func (t *TDSInt16) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSInt16(int16(byteOrder.Uint16(valBytes)))
	return nil
}

type TDSInt32 int32

func (t TDSInt32) Size() int {
	return 4
}

func (t *TDSInt32) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSInt32(int32(byteOrder.Uint32(valBytes)))
	return nil
}

type TDSInt64 int64

func (t TDSInt64) Size() int {
	return 8
}

func (t *TDSInt64) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSInt64(int64(byteOrder.Uint64(valBytes)))
	return nil
}

type TDSUint8 uint8

func (t TDSUint8) Value() uint32 {
	return 5
}

func (t TDSUint8) Size() int {
	return 1
}

func (t *TDSUint8) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSUint8(valBytes[0])
	return nil
}

type TDSUint16 uint16

func (t TDSUint16) Size() int {
	return 2
}

func (t *TDSUint16) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSUint16(byteOrder.Uint16(valBytes))
	return nil
}

type TDSUint32 uint32

func (t TDSUint32) Size() int {
	return 4
}

func (t *TDSUint32) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSUint32(byteOrder.Uint32(valBytes))
	return nil
}

type TDSUint64 uint64

func (t TDSUint64) Size() int {
	return 8
}

func (t *TDSUint64) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSUint64(byteOrder.Uint64(valBytes))
	return nil
}

type TDSFloat32 float32

func (t TDSFloat32) Size() int {
	return 4
}

func (t *TDSFloat32) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSFloat32(math.Float32frombits(byteOrder.Uint32(valBytes)))
	return nil
}

type TDSFloat64 float64

func (t TDSFloat64) Size() int {
	return 8
}

func (t *TDSFloat64) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSFloat64(math.Float64frombits(byteOrder.Uint64(valBytes)))
	return nil
}

type TDSFloat128 Float128

func (t TDSFloat128) Size() int {
	return 16
}

// While LabView can store extended precision float point numbers in various
// different numbers of bits – 80,, 64, 96, and 128 being the most popular (see
// https://www.ni.com/docs/en-US/bundle/labview/page/how-labview-stores-data-in-memory.html#d25545e113)
// – however when written to TDMS file, it is standardised as a 128-bit IEEE extended-precision format
// (https://www.ni.com/docs/en-US/bundle/labview/page/floating-point-numbers.html).
func (t *TDSFloat128) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, t.Size())
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSFloat128(parseQuad(valBytes, byteOrder))
	return nil
}

// Go's math.big.Float doesn't support NaN values.
type Float128 [16]byte

// Float64 converts the 128-bit extended precision float to a primitive float64.
// This loses a significant amount of precision. To avoid losing any precision
// at the cost of usability, see `BigFloat()`.
func (f Float128) Float64() float64 {
	return 0
}

func NewFloat128(value *big.Float) Float128 {
	return Float128{
		value: new(big.Float).Set(value),
		isNaN: false,
	}
}

func (f *Float128) IsNaN() bool {
	return f.isNaN
}

func (f *Float128) SetNaN() *Float128 {
	f.isNaN = true
	f.value = nil
	return f
}

func (f *Float128) SetValue(value *big.Float) *Float128 {
	f.isNaN = false
	f.value = new(big.Float).Set(value)
	return f
}

// GetValue returns the value of the Float128 as a big.Float. Although it
// returns a pointer, changing it does not change the Float128 itself. To do
// this, retrieve the big.Float, make any necessary changes and pass it back in
// via `SetValue()`.
func (f *Float128) GetValue() *big.Float {
	if f.isNaN {
		return nil
	}
	return new(big.Float).Set(f.value)
}

// parseQuad parses a 128-bit IEEE 754 quad precision float from 16 bytes.
// The bytes should be in the specified byte order (big-endian or little-endian).
func parseQuad(data []byte, order binary.ByteOrder) Float128 {
	if order == binary.LittleEndian {
		slices.Reverse(data)
	}

	// Extract sign bit (bit 127)
	sign := (data[0] >> 7) & 1

	// Extract exponent (bits 126-112, 15 bits total)
	exponent := uint16(data[0]&0x7F) << 8
	exponent |= uint16(data[1])

	// Extract mantissa (bits 111-0, 112 bits)
	mantissaBits := make([]byte, 14)
	copy(mantissaBits, data[2:16])

	// Quad precision has 113 bits of precision according to IEEE
	result := new(big.Float).SetPrec(113)

	// Handle special case of nan/inf
	if exponent == 0x7FFF {
		if isZeroMantissa(mantissaBits) {
			result.SetInf(sign == 1)
			return NewFloat128(result)
		} else {
			return *new(Float128).SetNaN()
		}
	}

	shiftAmount := new(big.Int).Lsh(big.NewInt(1), 112)

	if exponent == 0 {
		// Subnormal or zero
		if isZeroMantissa(mantissaBits) {
			result.SetInt64(0)
			return NewFloat128(result)
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

		return NewFloat128(result)
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

	return NewFloat128(result)
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

// The "with unit" data types are exactly the same, but just tell readers to
// exact the unit to be in a property called "unit_string".

type TDSFloat32WithUnit = TDSFloat32
type TDSFloat64WithUnit = TDSFloat64
type TDSFloat128WithUnit = TDSFloat128

type TDSString string

func (t TDSString) Size() int {
	return len(string(t))
}

func (t *TDSString) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	sizeBytes := make([]byte, 4)
	if _, err := reader.Read(sizeBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	size := int(byteOrder.Uint32(sizeBytes))

	data := make([]byte, size)
	if _, err := reader.Read(data); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSString(string(data))
	return nil
}

type TDSBool bool

func (t TDSBool) Size() int {
	return 1
}

func (t *TDSBool) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	boolBytes := make([]byte, 1)
	if _, err := reader.Read(boolBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSBool(boolBytes[0] != 0)
	return nil
}

type TDSTime struct {
	Timestamp int64
	Remainder uint64
}

func (t TDSTime) Size() int {
	return 16
}

// TDMS stores timestamps as a combination of i64 n# seconds since TDMS epoch
// which is 1st January 1904 at midnight and u64 number representing number
// fractional remainder, wherethe actual fractional n# seconds is retrieved by
// dividing by 2^-64. There is no timezone support.
// https://www.ni.com/en/support/documentation/supplemental/08/labview-timestamp-overview.html
func (t *TDSTime) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	timeBytes := make([]byte, 16)
	if _, err := reader.Read(timeBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	*t = TDSTime{
		Timestamp: int64(byteOrder.Uint64(timeBytes)),
		Remainder: byteOrder.Uint64(timeBytes[8:]),
	}
	return nil
}

// Time removes much of the precision in the TDS timestamp itself by converting
// from u64 remainder value (which is n# of 2^-64ths of a second =~ 0.05
// attoseconds) to nanoseconds. Thus, the TDS format retains approximately 1.8 ×
// 10^10 times more information than time.Time. This is not relevant for most
// purposes, but important to keep in mind.
func (t *TDSTime) Time() time.Time {
	// I'm not sure whether this big.Int stuff is necessary as opposed to doing
	// `float64(posFractions) * math.Pow(2, -64) * 1e9`. I need to experiment
	// with some large values to determine.
	ns := new(big.Int).SetUint64(t.Remainder)
	ns.Mul(ns, big.NewInt(1e9))
	ns.Rsh(ns, 64)
	return time.Unix(t.Timestamp, ns.Int64())
}

// The NI documentation provides nothing on how fixed points are stored. There
// is a page for how they are stored in memory while using LabVIEW, but not how
// it is stored on disk. Without an example or additional documentation, it's
// not possible to implement this. It's also not possible to know how large the
// data points are, which means you can't know how far to skip even if you want
// to ignore the fixed point channel. This means that the presence of any fixed
// point data renders a file unreadable. If you have more information or an
// actual TDMS file with a fixed point data channel in it, please contact the
// author of this repository so that this can be implemented.
// https://www.ni.com/docs/en-US/bundle/labview/page/numeric-data.html
// https://www.ni.com/docs/en-US/bundle/labview/page/numeric-data-types-table.html
// https://www.ni.com/docs/en-US/bundle/labview/page/labview-manager-data-types.html#d96127e328
type TDSFixedPoint struct{}

func (t TDSFixedPoint) Size() int {
	panic("not implemented")
}

func (t TDSFixedPoint) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	panic("not implemented")
}

type TDSComplexFloat32 complex64

func (t TDSComplexFloat32) Size() int {
	return 8
}

func (t *TDSComplexFloat32) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, 8)
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	real := math.Float32frombits(byteOrder.Uint32(valBytes))
	imag := math.Float32frombits(byteOrder.Uint32(valBytes))

	*t = TDSComplexFloat32(complex(real, imag))
	return nil
}

type TDSComplexFloat64 complex128

func (t TDSComplexFloat64) Size() int {
	return 16
}

func (t *TDSComplexFloat64) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	valBytes := make([]byte, 16)
	if _, err := reader.Read(valBytes); err != nil {
		return errors.Join(ErrReadFailed, err)
	}

	real := math.Float64frombits(byteOrder.Uint64(valBytes))
	imag := math.Float64frombits(byteOrder.Uint64(valBytes))

	*t = TDSComplexFloat64(complex(real, imag))
	return nil
}

// I'm not entirely sure, but I think data type of "DAQmx raw data" means that
// the actual data type is found inside the raw data index information, so
// "DAQmx" is not actually a data type itself but an indicator of a different
// representation of the data.
type TDSDAQmxRawData struct{}

func (t TDSDAQmxRawData) Size() int {
	return 0
}

func (t *TDSDAQmxRawData) Read(reader io.Reader, byteOrder binary.ByteOrder) error {
	return nil
}
