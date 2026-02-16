package tdms

import "fmt"

func ptr[T any](value T) *T {
	return &value
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
