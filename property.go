package tdms

import (
	"fmt"
	"time"
)

type Properties map[string]Property

func NewProperties() Properties {
	return map[string]Property{}
}

// GetInt returns the property value converted to an int64.
//
// If the value is numeric but not int64, e.g. int8, uint32, float64, etc., it
// will be converted to an int64.
//
// Returns ErrPropertyNotFound if the property is not found and no fallback is specified.
// Returns ErrIncorrectType if the property cannot be converted to an int64.
func (pr Properties) GetInt(name string, fallback ...int) (int, error) {
	prop, ok := pr[name]
	if !ok {
		if len(fallback) > 0 {
			return fallback[0], nil
		}
		return 0, ErrPropertyNotFound
	}
	return prop.ToInt()
}

// GetUint returns the property value converted to an uint64.
//
// If the value is numeric but not uint64, e.g. uint8, int32, float64, etc., it
// will be converted to an uint64.
//
// Returns ErrPropertyNotFound if the property is not found and no fallback is specified.
// Returns ErrIncorrectType if the property cannot be converted to an uint64.
func (pr Properties) GetUint(name string, fallback ...uint) (uint, error) {
	prop, ok := pr[name]
	if !ok {
		if len(fallback) > 0 {
			return fallback[0], nil
		}
		return 0, ErrPropertyNotFound
	}
	return prop.ToUint()
}

// GetFloat returns the property value converted to a float64.
//
// If the value is numeric but not float64, e.g. int32, uint8, etc., it
// will be converted to a float64.
//
// Returns ErrPropertyNotFound if the property is not found and no fallback is specified.
// Returns ErrIncorrectType if the property cannot be converted to a float64.
func (pr Properties) GetFloat(name string, fallback ...float64) (float64, error) {
	prop, ok := pr[name]
	if !ok {
		if len(fallback) > 0 {
			return fallback[0], nil
		}
		return 0, ErrPropertyNotFound
	}
	return prop.ToFloat()
}

// GetString returns the property value interpreted as a string.
//
// If the value is not a string, this will return an ErrIncorrectType error.
//
// Returns ErrPropertyNotFound if the property is not found and no fallback is specified.
// Returns ErrIncorrectType if the property is not a string type.
func (pr Properties) GetString(name string, fallback ...string) (string, error) {
	prop, ok := pr[name]
	if !ok {
		if len(fallback) > 0 {
			return fallback[0], nil
		}
		return "", ErrPropertyNotFound
	}
	return prop.AsString()
}

func (pr Properties) Add(name string, value any) Properties {
	prop := Property{
		Name:  name,
		Type:  DataType(0),
		Value: value,
	}

	switch v := value.(type) {
	case int:
		prop.Type = DataTypeInt64
		prop.Value = int64(v)
	case uint:
		prop.Type = DataTypeUint64
		prop.Value = uint64(v)
	case int8:
		prop.Type = DataTypeInt8
	case int16:
		prop.Type = DataTypeInt16
	case int32:
		prop.Type = DataTypeInt32
	case int64:
		prop.Type = DataTypeInt64
	case uint8:
		prop.Type = DataTypeUint8
	case uint16:
		prop.Type = DataTypeUint16
	case uint32:
		prop.Type = DataTypeUint32
	case uint64:
		prop.Type = DataTypeUint64
	case float32:
		prop.Type = DataTypeFloat32
	case float64:
		prop.Type = DataTypeFloat64
	case Float128:
		prop.Type = DataTypeFloat128
	case string:
		prop.Type = DataTypeString
	case bool:
		prop.Type = DataTypeBool
	case time.Time:
		prop.Type = DataTypeTimestamp
		prop.Value = TimeToTimestamp(v)
	case complex64:
		prop.Type = DataTypeComplex64
	case complex128:
		prop.Type = DataTypeComplex128
	default:
		panic(fmt.Sprintf("Unsupported property value type: %T", v))
	}

	pr[name] = prop
	return pr
}

// Property represents a key-value property attached to a file, group, or
// channel.
type Property struct {
	// Name is the name of this property.
	Name string

	// Type is the TDMS data type of the property value.
	Type DataType

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
	if p.Type != DataTypeInt8 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int8), nil
}

// AsInt16 returns the property value as an int16.
// Returns ErrIncorrectType if the property is not of type DataTypeInt16.
func (p Property) AsInt16() (int16, error) {
	if p.Type != DataTypeInt16 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int16), nil
}

// AsInt32 returns the property value as an int32.
// Returns ErrIncorrectType if the property is not of type DataTypeInt32.
func (p Property) AsInt32() (int32, error) {
	if p.Type != DataTypeInt32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int32), nil
}

// AsInt64 returns the property value as an int64.
// Returns ErrIncorrectType if the property is not of type DataTypeInt64.
func (p Property) AsInt64() (int64, error) {
	if p.Type != DataTypeInt64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(int64), nil
}

// AsUint8 returns the property value as a uint8.
// Returns ErrIncorrectType if the property is not of type DataTypeUint8.
func (p Property) AsUint8() (uint8, error) {
	if p.Type != DataTypeUint8 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint8), nil
}

// AsUint16 returns the property value as a uint16.
// Returns ErrIncorrectType if the property is not of type DataTypeUint16.
func (p Property) AsUint16() (uint16, error) {
	if p.Type != DataTypeUint16 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint16), nil
}

// AsUint32 returns the property value as a uint32.
// Returns ErrIncorrectType if the property is not of type DataTypeUint32.
func (p Property) AsUint32() (uint32, error) {
	if p.Type != DataTypeUint32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint32), nil
}

// AsUint64 returns the property value as a uint64.
// Returns ErrIncorrectType if the property is not of type DataTypeUint64.
func (p Property) AsUint64() (uint64, error) {
	if p.Type != DataTypeUint64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(uint64), nil
}

// AsFloat32 returns the property value as a float32.
// Returns ErrIncorrectType if the property is not of type DataTypeFloat32.
func (p Property) AsFloat32() (float32, error) {
	if p.Type != DataTypeFloat32 {
		return 0, ErrIncorrectType
	}
	return p.Value.(float32), nil
}

// AsFloat64 returns the property value as a float64.
// Returns ErrIncorrectType if the property is not of type DataTypeFloat64.
func (p Property) AsFloat64() (float64, error) {
	if p.Type != DataTypeFloat64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(float64), nil
}

// AsFloat128 returns the property value as a Float128.
// Returns ErrIncorrectType if the property is not of type DataTypeFloat128.
func (p Property) AsFloat128() (Float128, error) {
	if p.Type != DataTypeFloat128 {
		return Float128{}, ErrIncorrectType
	}
	return Float128(p.Value.(Float128)), nil
}

// AsString returns the property value as a string.
// Returns ErrIncorrectType if the property is not of type DataTypeString.
func (p Property) AsString() (string, error) {
	if p.Type != DataTypeString {
		return "", ErrIncorrectType
	}
	return p.Value.(string), nil
}

// AsBool returns the property value as a bool.
// Returns ErrIncorrectType if the property is not of type DataTypeBool.
func (p Property) AsBool() (bool, error) {
	if p.Type != DataTypeBool {
		return false, ErrIncorrectType
	}
	return p.Value.(bool), nil
}

// AsTimestamp returns the property value as a Timestamp.
// Returns ErrIncorrectType if the property is not of type DataTypeTimestamp.
func (p Property) AsTimestamp() (Timestamp, error) {
	if p.Type != DataTypeTimestamp {
		return Timestamp{}, ErrIncorrectType
	}
	return p.Value.(Timestamp), nil
}

// AsTime returns the property value as a time.Time, converting from the TDMS Timestamp format.
// Returns ErrIncorrectType if the property is not of type DataTypeTimestamp.
func (p Property) AsTime() (time.Time, error) {
	if p.Type != DataTypeTimestamp {
		return time.Time{}, ErrIncorrectType
	}

	t := p.Value.(Timestamp)
	return t.AsTime(), nil
}

// AsComplex64 returns the property value as a complex64.
// Returns ErrIncorrectType if the property is not of type DataTypeComplex64.
func (p Property) AsComplex64() (complex64, error) {
	if p.Type != DataTypeComplex64 {
		return 0, ErrIncorrectType
	}
	return p.Value.(complex64), nil
}

// AsComplex128 returns the property value as a complex128.
// Returns ErrIncorrectType if the property is not of type DataTypeComplex128.
func (p Property) AsComplex128() (complex128, error) {
	if p.Type != DataTypeComplex128 {
		return 0, ErrIncorrectType
	}
	return p.Value.(complex128), nil
}

// ToFloat converts the property value to a float64 value. This is different
// from the [AsFloat32], [AsFloat64], and [AsFloat128] methods, which don't do
// any conversion and will error if the type doesn't exactly match. Note that
// this may lose precision if e.g. the original value was an extended precision
// float.
//
// This method is useful if you need a float value and don't particularly care
// what type it was originally, as long as it can be converted to a float.
//
// If you call this method on a type which is incompatible with a float, it will
// produce an [ErrIncorrectType] error.
func (p Property) ToFloat() (float64, error) {
	switch v := p.Value.(type) {
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case Float128:
		return v.AsFloat64(), nil
	default:
		return 0, fmt.Errorf("%w: cannot convert %s to float64", ErrIncorrectType, p.Type)
	}
}

// ToInt converts the property value to an int value. This is different from the
// [AsInt8], [AsInt16], [AsInt32], and [AsInt64] methods, which don't do any
// conversion and will error if the type doesn't exactly match.
//
// This method is useful if you need a signed int value and don't particularly
// care what type it was originally, as long as it can be converted to a signed
// int.
//
// If you call this method on a type which is incompatible with a signed int, it
// will produce an [ErrIncorrectType] error.
func (p Property) ToInt() (int, error) {
	switch v := p.Value.(type) {
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case Float128:
		return int(v.AsFloat64()), nil
	default:
		return 0, fmt.Errorf("%w: cannot convert %s to int64", ErrIncorrectType, p.Type)
	}
}

// ToUint converts the property value to a uint value. This is different from
// the [AsUInt8], [AsUInt16], [AsUInt32], and [AsUInt64] methods, which don't do
// any conversion and will error if the type doesn't exactly match.
//
// This method is useful if you need an unsigned int value and don't
// particularly care what type it was originally, as long as it can be converted
// to an unsigned int.
//
// If you call this method on a type which is incompatible with an unsigned int,
// it will produce an [ErrIncorrectType] error.
func (p Property) ToUint() (uint, error) {
	switch v := p.Value.(type) {
	case uint8:
		return uint(v), nil
	case uint16:
		return uint(v), nil
	case uint32:
		return uint(v), nil
	case uint64:
		return uint(v), nil
	case int8:
		return uint(v), nil
	case int16:
		return uint(v), nil
	case int32:
		return uint(v), nil
	case int64:
		return uint(v), nil
	case float32:
		return uint(v), nil
	case float64:
		return uint(v), nil
	case Float128:
		return uint(v.AsFloat64()), nil
	default:
		return 0, fmt.Errorf("%w: cannot convert %s to uint64", ErrIncorrectType, p.Type)
	}
}
