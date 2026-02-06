package tdms

import "errors"

var (
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrReadFailed         = errors.New("failed to read data")
	ErrInvalidFileFormat  = errors.New("invalid file format")
	ErrInvalidPath        = errors.New("invalid object path")
	ErrUnsupportedType    = errors.New("unsupported data type")
	ErrIncorrectType      = errors.New("incorrect data type")
)
