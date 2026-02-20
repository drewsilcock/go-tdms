package tdms

import "fmt"

func ptr[T any](v T) *T {
	return &v
}

func allocateBuffer(dataType DataType, size int) (any, error) {
	switch dataType {
	case DataTypeInt8:
		return make([]int8, size), nil
	case DataTypeInt16:
		return make([]int16, size), nil
	case DataTypeInt32:
		return make([]int32, size), nil
	case DataTypeInt64:
		return make([]int64, size), nil
	case DataTypeUint8:
		return make([]uint8, size), nil
	case DataTypeUint16:
		return make([]uint16, size), nil
	case DataTypeUint32:
		return make([]uint32, size), nil
	case DataTypeUint64:
		return make([]uint64, size), nil
	case DataTypeFloat32, DataTypeFloat32WithUnit:
		return make([]float32, size), nil
	case DataTypeFloat64, DataTypeFloat64WithUnit:
		return make([]float64, size), nil
	case DataTypeFloat128, DataTypeFloat128WithUnit:
		return make([]Float128, size), nil
	case DataTypeString:
		return make([]string, size), nil
	case DataTypeBool:
		return make([]bool, size), nil
	case DataTypeTimestamp:
		return make([]Timestamp, size), nil
	case DataTypeComplex64:
		return make([]complex64, size), nil
	case DataTypeComplex128:
		return make([]complex128, size), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dataType)
	}
}

func bufferLen(buffer any) int {
	switch v := buffer.(type) {
	case []int8:
		return len(v)
	case []int16:
		return len(v)
	case []int32:
		return len(v)
	case []int64:
		return len(v)
	case []uint8:
		return len(v)
	case []uint16:
		return len(v)
	case []uint32:
		return len(v)
	case []uint64:
		return len(v)
	case []float32:
		return len(v)
	case []float64:
		return len(v)
	case []Float128:
		return len(v)
	case []string:
		return len(v)
	case []bool:
		return len(v)
	case []Timestamp:
		return len(v)
	case []complex64:
		return len(v)
	case []complex128:
		return len(v)
	case []any:
		return len(v)
	default:
		panic("unsupported buffer type")
	}
}

func sliceBuffer(buffer any, from, to int) any {
	switch v := buffer.(type) {
	case []int8:
		return v[from:to]
	case []int16:
		return v[from:to]
	case []int32:
		return v[from:to]
	case []int64:
		return v[from:to]
	case []uint8:
		return v[from:to]
	case []uint16:
		return v[from:to]
	case []uint32:
		return v[from:to]
	case []uint64:
		return v[from:to]
	case []float32:
		return v[from:to]
	case []float64:
		return v[from:to]
	case []Float128:
		return v[from:to]
	case []string:
		return v[from:to]
	case []bool:
		return v[from:to]
	case []Timestamp:
		return v[from:to]
	case []complex64:
		return v[from:to]
	case []complex128:
		return v[from:to]
	case []any:
		return v[from:to]
	default:
		panic("unsupported buffer type")
	}
}

// copySlice copies from input to output slice.
//
// Will panic if the output slice is not of the same type as the input type.
func copySlice(from, to any) error {
	switch f := from.(type) {
	case []int8:
		copy(to.([]int8), f)
	case []int16:
		copy(to.([]int16), f)
	case []int32:
		copy(to.([]int32), f)
	case []int64:
		copy(to.([]int64), f)
	case []uint8:
		copy(to.([]uint8), f)
	case []uint16:
		copy(to.([]uint16), f)
	case []uint32:
		copy(to.([]uint32), f)
	case []uint64:
		copy(to.([]uint64), f)
	case []float32:
		copy(to.([]float32), f)
	case []float64:
		copy(to.([]float64), f)
	case []Float128:
		copy(to.([]Float128), f)
	case []complex64:
		copy(to.([]complex64), f)
	case []complex128:
		copy(to.([]complex128), f)
	case []bool:
		copy(to.([]bool), f)
	case []string:
		copy(to.([]string), f)
	case []Timestamp:
		copy(to.([]Timestamp), f)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedType, from)
	}

	return nil
}
