package tdms

import (
	"errors"
	"fmt"
	"math/big"
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

// getNumericAccessor returns a closure that reads element i from a typed numeric
// slice and converts it to T. The type switch happens once per batch; subsequent
// per-element access is a direct indexed read with no reflection.
func getNumericAccessor[T Numeric](v any) func(int) T {
	switch s := v.(type) {
	case []int8:
		return func(i int) T { return T(s[i]) }
	case []int16:
		return func(i int) T { return T(s[i]) }
	case []int32:
		return func(i int) T { return T(s[i]) }
	case []int64:
		return func(i int) T { return T(s[i]) }
	case []uint8:
		return func(i int) T { return T(s[i]) }
	case []uint16:
		return func(i int) T { return T(s[i]) }
	case []uint32:
		return func(i int) T { return T(s[i]) }
	case []uint64:
		return func(i int) T { return T(s[i]) }
	case []float32:
		return func(i int) T { return T(s[i]) }
	case []float64:
		return func(i int) T { return T(s[i]) }
	default:
		return nil
	}
}

// getComplexAccessor returns a closure that reads element i from a typed slice
// and converts it to a complex type T. Numeric inputs are treated as the real
// component with zero imaginary part.
func getComplexAccessor[T complexFloat](v any) func(int) T {
	switch s := v.(type) {
	case []complex64:
		return func(i int) T { return T(s[i]) }
	case []complex128:
		return func(i int) T { return T(s[i]) }
	case []int8:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []int16:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []int32:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []int64:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []uint8:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []uint16:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []uint32:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []uint64:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []float32:
		return func(i int) T { return T(complex(float64(s[i]), 0)) }
	case []float64:
		return func(i int) T { return T(complex(s[i], 0)) }
	default:
		return nil
	}
}

// getBigFloatAccessor returns a closure that reads element i from a typed slice
// and converts it to *big.Float. Handles Float128 natively; other numeric types
// are converted via float64.
func getBigFloatAccessor(v any) func(int) *big.Float {
	switch s := v.(type) {
	case []Float128:
		return func(i int) *big.Float { return s[i].AsBigFloat() }
	case []int8:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []int16:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []int32:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []int64:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []uint8:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []uint16:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []uint32:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []uint64:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []float32:
		return func(i int) *big.Float { return big.NewFloat(float64(s[i])) }
	case []float64:
		return func(i int) *big.Float { return big.NewFloat(s[i]) }
	default:
		return nil
	}
}

// doNumericArithmetic performs element-wise add or subtract on two input slices,
// writing results to the pre-allocated output slice. Type conversion is handled
// via getNumericAccessor which does a single type-switch per call.
func doNumericArithmetic[T Numeric](out []T, left, right any, isAdd bool) error {
	lf := getNumericAccessor[T](left)
	rf := getNumericAccessor[T](right)
	if lf == nil || rf == nil {
		return fmt.Errorf("unsupported operand types for numeric arithmetic: %T, %T", left, right)
	}

	if isAdd {
		for i := range out {
			out[i] = lf(i) + rf(i)
		}
	} else {
		for i := range out {
			out[i] = lf(i) - rf(i)
		}
	}

	return nil
}

// doComplexArithmetic performs element-wise add or subtract on two input slices
// that promote to a complex type.
func doComplexArithmetic[T complexFloat](out []T, left, right any, isAdd bool) error {
	lf := getComplexAccessor[T](left)
	rf := getComplexAccessor[T](right)
	if lf == nil || rf == nil {
		return fmt.Errorf("unsupported operand types for complex arithmetic: %T, %T", left, right)
	}

	if isAdd {
		for i := range out {
			out[i] = lf(i) + rf(i)
		}
	} else {
		for i := range out {
			out[i] = lf(i) - rf(i)
		}
	}

	return nil
}

func makeArithmeticHandler(promotedType DataType, isAdd bool) opHandlerFunc {
	return func(leftValues any, rightValues any, output any) error {
		switch promotedType {
		case DataTypeInt8:
			return doNumericArithmetic(output.([]int8), leftValues, rightValues, isAdd)
		case DataTypeInt16:
			return doNumericArithmetic(output.([]int16), leftValues, rightValues, isAdd)
		case DataTypeInt32:
			return doNumericArithmetic(output.([]int32), leftValues, rightValues, isAdd)
		case DataTypeInt64:
			return doNumericArithmetic(output.([]int64), leftValues, rightValues, isAdd)
		case DataTypeUint8:
			return doNumericArithmetic(output.([]uint8), leftValues, rightValues, isAdd)
		case DataTypeUint16:
			return doNumericArithmetic(output.([]uint16), leftValues, rightValues, isAdd)
		case DataTypeUint32:
			return doNumericArithmetic(output.([]uint32), leftValues, rightValues, isAdd)
		case DataTypeUint64:
			return doNumericArithmetic(output.([]uint64), leftValues, rightValues, isAdd)
		case DataTypeFloat32, DataTypeFloat32WithUnit:
			return doNumericArithmetic(output.([]float32), leftValues, rightValues, isAdd)
		case DataTypeFloat64, DataTypeFloat64WithUnit:
			return doNumericArithmetic(output.([]float64), leftValues, rightValues, isAdd)
		case DataTypeFloat128, DataTypeFloat128WithUnit:
			out := output.([]Float128)
			lf := getBigFloatAccessor(leftValues)
			rf := getBigFloatAccessor(rightValues)
			if lf == nil || rf == nil {
				return fmt.Errorf("unsupported operand types for Float128 arithmetic: %T, %T", leftValues, rightValues)
			}
			for i := range out {
				l := lf(i)
				r := rf(i)
				var result *big.Float
				if isAdd {
					result = new(big.Float).Add(l, r)
				} else {
					result = new(big.Float).Sub(l, r)
				}
				out[i] = *new(Float128).SetBigFloat(result)
			}
			return nil
		case DataTypeComplex64:
			return doComplexArithmetic(output.([]complex64), leftValues, rightValues, isAdd)
		case DataTypeComplex128:
			return doComplexArithmetic(output.([]complex128), leftValues, rightValues, isAdd)
		default:
			return fmt.Errorf("unsupported promoted type: %s", promotedType)
		}
	}
}

type complexFloat interface{ complex64 | complex128 }

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
