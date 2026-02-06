package tdms

import (
	"fmt"
	"time"
)

// Property represents a key-value property attached to a file, group, or
// channel.
type Property struct {
	// Name is the name of this property.
	Name string

	// TypeCode is the TDMS data type of the property value.
	TypeCode DataType

	// Value is the actual property value. Use the As* methods or a type switch
	// in your own code to safely extract the value as a specific type.
	Value any
}

// String implements [fmt.Stringer] interface, returning the string
// representation of the key and value.
func (p Property) String() string {
	return fmt.Sprintf("%s: %v", p.Name, p.Value)
}

// AsInt8 returns the property value as an int8.
// Returns ErrIncorrectType if the property is not of type DataTypeInt8.
func (p Property) AsInt8() (int8, error) {
	if p.TypeCode != DataTypeInt8 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int8), nil
}

// AsInt16 returns the property value as an int16.
// Returns ErrIncorrectType if the property is not of type DataTypeInt16.
func (p Property) AsInt16() (int16, error) {
	if p.TypeCode != DataTypeInt16 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int16), nil
}

// AsInt32 returns the property value as an int32.
// Returns ErrIncorrectType if the property is not of type DataTypeInt32.
func (p Property) AsInt32() (int32, error) {
	if p.TypeCode != DataTypeInt32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int32), nil
}

// AsInt64 returns the property value as an int64.
// Returns ErrIncorrectType if the property is not of type DataTypeInt64.
func (p Property) AsInt64() (int64, error) {
	if p.TypeCode != DataTypeInt64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int64), nil
}

// AsUint8 returns the property value as a uint8.
// Returns ErrIncorrectType if the property is not of type DataTypeUint8.
func (p Property) AsUint8() (uint8, error) {
	if p.TypeCode != DataTypeUint8 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint8), nil
}

// AsUint16 returns the property value as a uint16.
// Returns ErrIncorrectType if the property is not of type DataTypeUint16.
func (p Property) AsUint16() (uint16, error) {
	if p.TypeCode != DataTypeUint16 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint16), nil
}

// AsUint32 returns the property value as a uint32.
// Returns ErrIncorrectType if the property is not of type DataTypeUint32.
func (p Property) AsUint32() (uint32, error) {
	if p.TypeCode != DataTypeUint32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint32), nil
}

// AsUint64 returns the property value as a uint64.
// Returns ErrIncorrectType if the property is not of type DataTypeUint64.
func (p Property) AsUint64() (uint64, error) {
	if p.TypeCode != DataTypeUint64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint64), nil
}

// AsFloat32 returns the property value as a float32.
// Returns ErrIncorrectType if the property is not of type DataTypeFloat32.
func (p Property) AsFloat32() (float32, error) {
	if p.TypeCode != DataTypeFloat32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(float32), nil
}

// AsFloat64 returns the property value as a float64.
// Returns ErrIncorrectType if the property is not of type DataTypeFloat64.
func (p Property) AsFloat64() (float64, error) {
	if p.TypeCode != DataTypeFloat64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(float64), nil
}

// AsFloat128 returns the property value as a Float128.
// Returns ErrIncorrectType if the property is not of type DataTypeFloat128.
func (p Property) AsFloat128() (Float128, error) {
	if p.TypeCode != DataTypeFloat128 {
		return Float128{}, ErrIncorrectType
	}
	return Float128(p.Value.(Float128)), nil
}

// AsString returns the property value as a string.
// Returns ErrIncorrectType if the property is not of type DataTypeString.
func (p Property) AsString() (string, error) {
	if p.TypeCode != DataTypeString {
		return "", ErrIncorrectType
	}
	return p.Value.(string), nil
}

// AsBool returns the property value as a bool.
// Returns ErrIncorrectType if the property is not of type DataTypeBool.
func (p Property) AsBool() (bool, error) {
	if p.TypeCode != DataTypeBool {
		return false, ErrIncorrectType
	}
	return p.Value.(bool), nil
}

// AsTimestamp returns the property value as a Timestamp.
// Returns ErrIncorrectType if the property is not of type DataTypeTimestamp.
func (p Property) AsTimestamp() (Timestamp, error) {
	if p.TypeCode != DataTypeTimestamp {
		return Timestamp{}, ErrIncorrectType
	}
	return p.Value.(Timestamp), nil
}

// AsTime returns the property value as a time.Time, converting from the TDMS Timestamp format.
// Returns ErrIncorrectType if the property is not of type DataTypeTimestamp.
func (p Property) AsTime() (time.Time, error) {
	if p.TypeCode != DataTypeTimestamp {
		return time.Time{}, ErrIncorrectType
	}

	t := p.Value.(Timestamp)
	return t.AsTime(), nil
}

// AsComplex64 returns the property value as a complex64.
// Returns ErrIncorrectType if the property is not of type DataTypeComplex64.
func (p Property) AsComplex64() (complex64, error) {
	if p.TypeCode != DataTypeComplex64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(complex64), nil
}

// AsComplex128 returns the property value as a complex128.
// Returns ErrIncorrectType if the property is not of type DataTypeComplex128.
func (p Property) AsComplex128() (complex128, error) {
	if p.TypeCode != DataTypeComplex128 {
		return 0, ErrIncorrectType
	}
	return p.Value.(complex128), nil
}
