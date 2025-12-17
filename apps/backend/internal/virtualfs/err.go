package virtualfs

import "errors"

var (
	ErrNodeNotFound               = errors.New("node not found")
	ErrNodeConflict               = errors.New("node conflict")
	ErrUnsupportedOperation       = errors.New("unsupported operation")
	ErrCursorMismatchedOrderField = errors.New("cursor mismatched order field")
	ErrCursorMismatchedDirection  = errors.New("cursor mismatched direction")
)
