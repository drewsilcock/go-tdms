package tdms

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

// Type promotion, a la:
// https://numpy.org/devdocs/reference/arrays.promotion.html#arrays-promotion
//
// Rules:
//
//   - If either type is a float, the type gets promoted to a float.
//   - If one type has higher precision (e.g. float64 vs. flaot32), the higher precision wins.
//   - You can't add bools or timestamps anything else, including their own type.
//   - You can add complex numbers to other numerics – if the other numeric is not
//     a complex type, it's assumed to be the real component.
//   - If one type is signed and the other is not, type gets promoted to signed.
//   - You can add strings to other strings only.
//
// As such, the hierarchy of promotion is:
//
// 1. Unsigned int
// 2. Signed int
// 3. Float
// 4. Complex
//
// With precision forming a hierarchy within each type.
func getPromotedType(leftType DataType, rightType DataType) (DataType, error) {
	if leftType == DataTypeBool || rightType == DataTypeBool {
		return DataTypeVoid, errors.New("cannot perform arithmetic on boolean types")
	}

	if leftType == DataTypeTimestamp || rightType == DataTypeTimestamp {
		return DataTypeVoid, errors.New("cannot perform arithmetic on timestamp types")
	}

	if leftType == DataTypeString || rightType == DataTypeString {
		return DataTypeVoid, errors.New("cannot perform arithmetic on string types")
	}

	isComplex := func(dt DataType) bool {
		return dt == DataTypeComplex64 || dt == DataTypeComplex128
	}

	isFloat := func(dt DataType) bool {
		return dt == DataTypeFloat32 || dt == DataTypeFloat64 || dt == DataTypeFloat128 ||
			dt == DataTypeFloat32WithUnit || dt == DataTypeFloat64WithUnit || dt == DataTypeFloat128WithUnit
	}

	isSigned := func(dt DataType) bool {
		return dt == DataTypeInt8 || dt == DataTypeInt16 || dt == DataTypeInt32 || dt == DataTypeInt64
	}

	isUnsigned := func(dt DataType) bool {
		return dt == DataTypeUint8 || dt == DataTypeUint16 || dt == DataTypeUint32 || dt == DataTypeUint64
	}

	maxBits := 0

	for _, dt := range []DataType{leftType, rightType} {
		switch dt {
		case DataTypeUint8, DataTypeInt8:
			maxBits = max(maxBits, 8)
		case DataTypeUint16, DataTypeInt16:
			maxBits = max(maxBits, 16)
		case DataTypeUint32, DataTypeInt32:
			maxBits = max(maxBits, 32)
		case DataTypeUint64, DataTypeInt64:
			maxBits = max(maxBits, 64)
		case DataTypeFloat32, DataTypeFloat32WithUnit:
			maxBits = max(maxBits, 32)
		case DataTypeFloat64, DataTypeFloat64WithUnit, DataTypeComplex64:
			maxBits = max(maxBits, 64)
		case DataTypeFloat128, DataTypeFloat128WithUnit, DataTypeComplex128:
			maxBits = max(maxBits, 128)
		}
	}

	if isComplex(leftType) || isComplex(rightType) {
		if maxBits == 128 {
			return DataTypeComplex128, nil
		}

		return DataTypeComplex64, nil
	}

	if isFloat(leftType) || isFloat(rightType) {
		switch maxBits {
		case 32:
			return DataTypeFloat32, nil
		case 64:
			return DataTypeFloat64, nil
		case 128:
			return DataTypeFloat128, nil
		default:
			return DataTypeVoid, fmt.Errorf("unsupported float bit size: %d", maxBits)
		}
	}

	if isSigned(leftType) && isSigned(rightType) {
		switch maxBits {
		case 8:
			return DataTypeInt8, nil
		case 16:
			return DataTypeInt16, nil
		case 32:
			return DataTypeInt32, nil
		case 64:
			return DataTypeInt64, nil
		default:
			return DataTypeVoid, fmt.Errorf("unsupported signed integer bit size: %d", maxBits)
		}
	}

	if isSigned(leftType) || isSigned(rightType) {
		// If only one is signed and the other unsigned, we need to bump up the
		// n# bits to avoid losing precision.

		switch maxBits {
		case 8:
			return DataTypeInt16, nil
		case 16:
			return DataTypeInt32, nil
		case 32:
			return DataTypeInt64, nil
		case 64:
			// There's no int128 so we need to move to float.
			return DataTypeFloat64, nil
		default:
			return DataTypeVoid, fmt.Errorf("unsupported signed integer bit size: %d", maxBits)
		}
	}

	if isUnsigned(leftType) || isUnsigned(rightType) {
		switch maxBits {
		case 8:
			return DataTypeUint8, nil
		case 16:
			return DataTypeUint16, nil
		case 32:
			return DataTypeUint32, nil
		case 64:
			return DataTypeUint64, nil
		default:
			return DataTypeVoid, fmt.Errorf("unsupported unsigned integer bit size: %d", maxBits)
		}
	}

	return DataTypeVoid, fmt.Errorf("unknown types: %s, %s", leftType, rightType)
}

type opHandlerFunc func(leftValues any, rightValues any, output any) error
type opTypeKey struct{ leftType, rightType DataType }

var addHandlers = map[opTypeKey]opHandlerFunc{}
var subHandlers = map[opTypeKey]opHandlerFunc{}

func registerHandler(opMap map[opTypeKey]opHandlerFunc, leftType, rightType DataType, handler opHandlerFunc) {
	opMap[opTypeKey{leftType, rightType}] = handler
}

func init() {
	// Register handlers for all type combinations
	numericTypes := []DataType{
		DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64,
		DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64,
		DataTypeFloat32, DataTypeFloat64, DataTypeFloat128,
		DataTypeFloat32WithUnit, DataTypeFloat64WithUnit, DataTypeFloat128WithUnit,
		DataTypeComplex64, DataTypeComplex128,
	}

	for _, leftType := range numericTypes {
		for _, rightType := range numericTypes {
			promotedType, err := getPromotedType(leftType, rightType)
			if err != nil {
				continue
			}

			registerHandler(addHandlers, leftType, rightType, makeArithmeticHandler(promotedType, true))
			registerHandler(subHandlers, leftType, rightType, makeArithmeticHandler(promotedType, false))
		}
	}
}

func makeArithmeticHandler(promotedType DataType, isAdd bool) opHandlerFunc {
	return func(leftValues any, rightValues any, output any) error {
		leftVal := reflect.ValueOf(leftValues)
		rightVal := reflect.ValueOf(rightValues)

		if leftVal.Len() == 0 {
			return nil
		}

		// Handle different promoted types
		switch promotedType {
		case DataTypeInt8:
			out := output.([]int8)
			for i := range leftVal.Len() {
				l := convertToSignedInteger[int8](leftVal.Index(i))
				r := convertToSignedInteger[int8](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeInt16:
			out := output.([]int16)
			for i := range leftVal.Len() {
				l := convertToSignedInteger[int16](leftVal.Index(i))
				r := convertToSignedInteger[int16](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeInt32:
			out := output.([]int32)
			for i := range leftVal.Len() {
				l := convertToSignedInteger[int32](leftVal.Index(i))
				r := convertToSignedInteger[int32](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeInt64:
			out := output.([]int64)
			for i := range leftVal.Len() {
				l := convertToSignedInteger[int64](leftVal.Index(i))
				r := convertToSignedInteger[int64](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint8:
			out := output.([]uint8)
			for i := range leftVal.Len() {
				l := convertToUnsignedInteger[uint8](leftVal.Index(i))
				r := convertToUnsignedInteger[uint8](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint16:
			out := output.([]uint16)
			for i := range leftVal.Len() {
				l := convertToUnsignedInteger[uint16](leftVal.Index(i))
				r := convertToUnsignedInteger[uint16](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint32:
			out := output.([]uint32)
			for i := range leftVal.Len() {
				l := convertToUnsignedInteger[uint32](leftVal.Index(i))
				r := convertToUnsignedInteger[uint32](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint64:
			out := output.([]uint64)
			for i := range leftVal.Len() {
				l := convertToUnsignedInteger[uint64](leftVal.Index(i))
				r := convertToUnsignedInteger[uint64](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeFloat32, DataTypeFloat32WithUnit:
			out := output.([]float32)
			for i := range leftVal.Len() {
				l := convertToFloat[float32](leftVal.Index(i))
				r := convertToFloat[float32](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeFloat64, DataTypeFloat64WithUnit:
			out := output.([]float64)
			for i := range leftVal.Len() {
				l := convertToFloat[float64](leftVal.Index(i))
				r := convertToFloat[float64](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeFloat128, DataTypeFloat128WithUnit:
			out := output.([]Float128)
			for i := range leftVal.Len() {
				l := convertToBigFloat(leftVal.Index(i))
				r := convertToBigFloat(rightVal.Index(i))
				var result *big.Float
				if isAdd {
					result = new(big.Float).Add(l, r)
				} else {
					result = new(big.Float).Sub(l, r)
				}
				out[i] = *new(Float128).SetBigFloat(result)
			}

		case DataTypeComplex64:
			out := output.([]complex64)
			for i := range leftVal.Len() {
				l := convertToComplex[complex64](leftVal.Index(i))
				r := convertToComplex[complex64](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeComplex128:
			out := output.([]complex128)
			for i := range leftVal.Len() {
				l := convertToComplex[complex128](leftVal.Index(i))
				r := convertToComplex[complex128](rightVal.Index(i))
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		default:
			return fmt.Errorf("unsupported promoted type: %s", promotedType)
		}

		return nil
	}
}

type signedInteger interface{ int8 | int16 | int32 | int64 }
type unsignedInteger interface {
	uint8 | uint16 | uint32 | uint64
}
type float interface{ float32 | float64 }
type complexFloat interface{ complex64 | complex128 }

func convertToUnsignedInteger[T unsignedInteger](v reflect.Value) T {
	if v.CanUint() {
		return T(v.Uint())
	}

	if v.CanInt() {
		return T(v.Int())
	}

	return 0
}

func convertToSignedInteger[T signedInteger](v reflect.Value) T {
	if v.CanInt() {
		return T(v.Int())
	}

	if v.CanUint() {
		return T(v.Uint())
	}

	return 0
}

func convertToFloat[T float](v reflect.Value) T {
	if v.CanFloat() {
		return T(v.Float())
	}

	if v.CanInt() {
		return T(v.Int())
	}

	if v.CanUint() {
		return T(v.Uint())
	}

	return 0
}

func convertToComplex[T complexFloat](v reflect.Value) T {
	if v.CanComplex() {
		return T(v.Complex())
	}

	if v.CanFloat() {
		return T(complex(v.Float(), 0))
	}

	if v.CanInt() {
		return T(complex(float64(v.Int()), 0))
	}

	if v.CanUint() {
		return T(complex(float64(v.Uint()), 0))
	}

	return 0
}

func convertToBigFloat(v reflect.Value) *big.Float {
	if f, ok := v.Interface().(Float128); ok {
		return f.AsBigFloat()
	}

	return big.NewFloat(convertToFloat[float64](v))
}

func getDataType(v any) DataType {
	switch v.(type) {
	case []int8:
		return DataTypeInt8
	case []int16:
		return DataTypeInt16
	case []int32:
		return DataTypeInt32
	case []int64:
		return DataTypeInt64
	case []uint8:
		return DataTypeUint8
	case []uint16:
		return DataTypeUint16
	case []uint32:
		return DataTypeUint32
	case []uint64:
		return DataTypeUint64
	case []float32:
		return DataTypeFloat32
	case []float64:
		return DataTypeFloat64
	case []Float128:
		return DataTypeFloat128
	case []complex64:
		return DataTypeComplex64
	case []complex128:
		return DataTypeComplex128
	case []bool:
		return DataTypeBool
	case []string:
		return DataTypeString
	case []Timestamp:
		return DataTypeTimestamp
	default:
		return DataTypeVoid
	}
}

func typePromotedAdd(leftValues any, rightValues any, output any) error {
	leftType := getDataType(leftValues)
	rightType := getDataType(rightValues)

	if leftType == DataTypeVoid || rightType == DataTypeVoid {
		return ErrIncorrectType
	}

	handler, ok := addHandlers[opTypeKey{leftType, rightType}]
	if !ok {
		return fmt.Errorf("no add handler for types %s and %s", leftType, rightType)
	}

	return handler(leftValues, rightValues, output)
}

func typePromotedSubtract(leftValues any, rightValues any, output any) error {
	leftType := getDataType(leftValues)
	rightType := getDataType(rightValues)

	if leftType == DataTypeVoid || rightType == DataTypeVoid {
		return ErrIncorrectType
	}

	handler, ok := subHandlers[opTypeKey{leftType, rightType}]
	if !ok {
		return fmt.Errorf("no subtract handler for types %s and %s", leftType, rightType)
	}

	return handler(leftValues, rightValues, output)
}
