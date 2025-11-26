package blob

import "errors"

var (
	ErrConflict = errors.New("key already used for a different blob")
	ErrNotFound = errors.New("key not found")
)
