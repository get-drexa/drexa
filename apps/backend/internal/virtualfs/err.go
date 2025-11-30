package virtualfs

import "errors"

var (
	ErrNodeNotFound         = errors.New("node not found")
	ErrNodeConflict         = errors.New("node conflict")
	ErrUnsupportedOperation = errors.New("unsupported operation")
)
