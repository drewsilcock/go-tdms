package tdms

import "errors"

var (
	// ErrUnsupportedVersion indicates that the TDMS file uses a version not supported by this library.
	ErrUnsupportedVersion = errors.New("unsupported version")

	// ErrReadFailed indicates that reading data from the underlying file or reader failed.
	ErrReadFailed = errors.New("failed to read data")

	// ErrInvalidFileFormat indicates that the TDMS file structure is malformed or doesn't conform to the specification.
	ErrInvalidFileFormat = errors.New("invalid file format")

	// ErrInvalidPath indicates that an object path within the TDMS file is not properly formatted.
	ErrInvalidPath = errors.New("invalid object path")

	// ErrUnsupportedType indicates that the data type encountered is not supported by this library.
	ErrUnsupportedType = errors.New("unsupported data type")

	// ErrIncorrectType indicates that a type assertion or conversion failed because the actual type differs from the expected type.
	ErrIncorrectType = errors.New("incorrect data type")
)
