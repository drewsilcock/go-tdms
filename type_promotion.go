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
		return DataTypeVoid, fmt.Errorf("cannot perform arithmetic on boolean types")
	}

	if leftType == DataTypeTimestamp || rightType == DataTypeTimestamp {
		return DataTypeVoid, fmt.Errorf("cannot perform arithmetic on timestamp types")
	}

	if leftType == DataTypeString || rightType == DataTypeString {
		if leftType != rightType {
			return DataTypeVoid, errors.New("cannot perform arithmetic on string and non-string types")
		}

		return DataTypeString, nil
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

// Helper to convert any numeric value to float64 for complex operations
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case Float128:
		return val.AsFloat64()
	default:
		return 0
	}
}

// Helper to convert any numeric value to big.Float
func toBigFloat(v any) *big.Float {
	switch val := v.(type) {
	case int8:
		return big.NewFloat(float64(val))
	case int16:
		return big.NewFloat(float64(val))
	case int32:
		return big.NewFloat(float64(val))
	case int64:
		return big.NewFloat(float64(val))
	case uint8:
		return big.NewFloat(float64(val))
	case uint16:
		return big.NewFloat(float64(val))
	case uint32:
		return big.NewFloat(float64(val))
	case uint64:
		return big.NewFloat(float64(val))
	case float32:
		return big.NewFloat(float64(val))
	case float64:
		return big.NewFloat(val)
	case Float128:
		return val.AsBigFloat()
	default:
		return big.NewFloat(0)
	}
}

func init() {
	// Register handlers for all type combinations
	numericTypes := []DataType{
		DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64,
		DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64,
		DataTypeFloat32, DataTypeFloat64, DataTypeFloat128,
		DataTypeComplex64, DataTypeComplex128,
	}

	for _, leftType := range numericTypes {
		for _, rightType := range numericTypes {
			promotedType, err := getPromotedType(leftType, rightType)
			if err != nil {
				continue
			}

			// Register add handler
			registerHandler(addHandlers, leftType, rightType, makeArithmeticHandler(leftType, rightType, promotedType, true))

			// Register subtract handler
			registerHandler(subHandlers, leftType, rightType, makeArithmeticHandler(leftType, rightType, promotedType, false))
		}
	}

	// Special handlers for strings (add only - concatenation)
	registerHandler(addHandlers, DataTypeString, DataTypeString, func(leftValues any, rightValues any, output any) error {
		left := leftValues.([]string)
		right := rightValues.([]string)
		out := output.([]string)
		for i := range left {
			out[i] = left[i] + right[i]
		}
		return nil
	})
}

func makeArithmeticHandler(leftType, rightType, promotedType DataType, isAdd bool) opHandlerFunc {
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
			for i := 0; i < leftVal.Len(); i++ {
				l := int8(leftVal.Index(i).Int())
				r := int8(rightVal.Index(i).Int())
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeInt16:
			out := output.([]int16)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToInt16(leftVal.Index(i), leftType)
				r := convertToInt16(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeInt32:
			out := output.([]int32)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToInt32(leftVal.Index(i), leftType)
				r := convertToInt32(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeInt64:
			out := output.([]int64)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToInt64(leftVal.Index(i), leftType)
				r := convertToInt64(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint8:
			out := output.([]uint8)
			for i := 0; i < leftVal.Len(); i++ {
				l := uint8(leftVal.Index(i).Uint())
				r := uint8(rightVal.Index(i).Uint())
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint16:
			out := output.([]uint16)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToUint16(leftVal.Index(i), leftType)
				r := convertToUint16(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint32:
			out := output.([]uint32)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToUint32(leftVal.Index(i), leftType)
				r := convertToUint32(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeUint64:
			out := output.([]uint64)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToUint64(leftVal.Index(i), leftType)
				r := convertToUint64(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeFloat32:
			out := output.([]float32)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToFloat32(leftVal.Index(i), leftType)
				r := convertToFloat32(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeFloat64:
			out := output.([]float64)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToFloat64(leftVal.Index(i), leftType)
				r := convertToFloat64(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeFloat128:
			out := output.([]Float128)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToBigFloat(leftVal.Index(i), leftType)
				r := convertToBigFloat(rightVal.Index(i), rightType)
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
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToComplex64(leftVal.Index(i), leftType)
				r := convertToComplex64(rightVal.Index(i), rightType)
				if isAdd {
					out[i] = l + r
				} else {
					out[i] = l - r
				}
			}

		case DataTypeComplex128:
			out := output.([]complex128)
			for i := 0; i < leftVal.Len(); i++ {
				l := convertToComplex128(leftVal.Index(i), leftType)
				r := convertToComplex128(rightVal.Index(i), rightType)
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

// Conversion helper functions
func convertToInt16(v reflect.Value, srcType DataType) int16 {
	switch srcType {
	case DataTypeInt8:
		return int16(v.Int())
	case DataTypeInt16:
		return int16(v.Int())
	case DataTypeUint8:
		return int16(v.Uint())
	default:
		return 0
	}
}

func convertToInt32(v reflect.Value, srcType DataType) int32 {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32:
		return int32(v.Int())
	case DataTypeUint8, DataTypeUint16:
		return int32(v.Uint())
	default:
		return 0
	}
}

func convertToInt64(v reflect.Value, srcType DataType) int64 {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64:
		return v.Int()
	case DataTypeUint8, DataTypeUint16, DataTypeUint32:
		return int64(v.Uint())
	default:
		return 0
	}
}

func convertToUint16(v reflect.Value, srcType DataType) uint16 {
	switch srcType {
	case DataTypeUint8:
		return uint16(v.Uint())
	case DataTypeUint16:
		return uint16(v.Uint())
	default:
		return 0
	}
}

func convertToUint32(v reflect.Value, srcType DataType) uint32 {
	switch srcType {
	case DataTypeUint8, DataTypeUint16, DataTypeUint32:
		return uint32(v.Uint())
	default:
		return 0
	}
}

func convertToUint64(v reflect.Value, srcType DataType) uint64 {
	switch srcType {
	case DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64:
		return v.Uint()
	default:
		return 0
	}
}

func convertToFloat32(v reflect.Value, srcType DataType) float32 {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64:
		return float32(v.Int())
	case DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64:
		return float32(v.Uint())
	case DataTypeFloat32:
		return float32(v.Float())
	default:
		return 0
	}
}

func convertToFloat64(v reflect.Value, srcType DataType) float64 {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64:
		return float64(v.Int())
	case DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64:
		return float64(v.Uint())
	case DataTypeFloat32, DataTypeFloat64:
		return v.Float()
	default:
		return 0
	}
}

func convertToBigFloat(v reflect.Value, srcType DataType) *big.Float {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64:
		return big.NewFloat(float64(v.Int()))
	case DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64:
		return big.NewFloat(float64(v.Uint()))
	case DataTypeFloat32, DataTypeFloat64:
		return big.NewFloat(v.Float())
	case DataTypeFloat128:
		return v.Interface().(Float128).AsBigFloat()
	default:
		return big.NewFloat(0)
	}
}

func convertToComplex64(v reflect.Value, srcType DataType) complex64 {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64:
		return complex64(complex(float64(v.Int()), 0))
	case DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64:
		return complex64(complex(float64(v.Uint()), 0))
	case DataTypeFloat32, DataTypeFloat64:
		return complex64(complex(v.Float(), 0))
	case DataTypeFloat128:
		f := v.Interface().(Float128).AsFloat64()
		return complex64(complex(f, 0))
	case DataTypeComplex64:
		return complex64(v.Complex())
	case DataTypeComplex128:
		return complex64(v.Complex())
	default:
		return 0
	}
}

func convertToComplex128(v reflect.Value, srcType DataType) complex128 {
	switch srcType {
	case DataTypeInt8, DataTypeInt16, DataTypeInt32, DataTypeInt64:
		return complex(float64(v.Int()), 0)
	case DataTypeUint8, DataTypeUint16, DataTypeUint32, DataTypeUint64:
		return complex(float64(v.Uint()), 0)
	case DataTypeFloat32, DataTypeFloat64:
		return complex(v.Float(), 0)
	case DataTypeFloat128:
		f := v.Interface().(Float128).AsFloat64()
		return complex(f, 0)
	case DataTypeComplex64:
		return complex128(v.Complex())
	case DataTypeComplex128:
		return v.Complex()
	default:
		return 0
	}
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
