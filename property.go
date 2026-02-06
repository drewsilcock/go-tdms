package tdms

import (
	"fmt"
	"time"
)

// Property represents a key-value property attached to a file, group, or
// channel.
type Property struct {
	Name     string
	TypeCode DataType
	Value    any
}

// String implements Stringer interface to allow property to be printed.
func (p Property) String() string {
	return fmt.Sprintf("%s: %v", p.Name, p.Value)
}

func (p Property) AsInt8() (int8, error) {
	if p.TypeCode != DataTypeInt8 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int8), nil
}

func (p Property) AsInt16() (int16, error) {
	if p.TypeCode != DataTypeInt16 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int16), nil
}

func (p Property) AsInt32() (int32, error) {
	if p.TypeCode != DataTypeInt32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int32), nil
}

func (p Property) AsInt64() (int64, error) {
	if p.TypeCode != DataTypeInt64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int64), nil
}

func (p Property) AsUint8() (uint8, error) {
	if p.TypeCode != DataTypeUint8 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint8), nil
}

func (p Property) AsUint16() (uint16, error) {
	if p.TypeCode != DataTypeUint16 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint16), nil
}

func (p Property) AsUint32() (uint32, error) {
	if p.TypeCode != DataTypeUint32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint32), nil
}

func (p Property) AsUint64() (uint64, error) {
	if p.TypeCode != DataTypeUint64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint64), nil
}

func (p Property) AsFloat32() (float32, error) {
	if p.TypeCode != DataTypeFloat32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(float32), nil
}

func (p Property) AsFloat64() (float64, error) {
	if p.TypeCode != DataTypeFloat64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(float64), nil
}

func (p Property) AsFloat128() (Float128, error) {
	if p.TypeCode != DataTypeFloat128 {
		return Float128{}, ErrIncorrectType
	}
	return Float128(p.Value.(Float128)), nil
}

func (p Property) AsString() (string, error) {
	if p.TypeCode != DataTypeString {
		return "", ErrIncorrectType
	}
	return p.Value.(string), nil
}

func (p Property) AsBool() (bool, error) {
	if p.TypeCode != DataTypeBool {
		return false, ErrIncorrectType
	}
	return p.Value.(bool), nil
}

func (p Property) AsTimestamp() (Timestamp, error) {
	if p.TypeCode != DataTypeTimestamp {
		return Timestamp{}, ErrIncorrectType
	}
	return p.Value.(Timestamp), nil
}

func (p Property) AsTime() (time.Time, error) {
	if p.TypeCode != DataTypeTimestamp {
		return time.Time{}, ErrIncorrectType
	}

	t := p.Value.(Timestamp)
	return t.AsTime(), nil
}

func (p Property) AsComplex64() (complex64, error) {
	if p.TypeCode != DataTypeComplex64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(complex64), nil
}

func (p Property) AsComplex128() (complex128, error) {
	if p.TypeCode != DataTypeComplex128 {
		return 0, ErrIncorrectType
	}
	return p.Value.(complex128), nil
}
