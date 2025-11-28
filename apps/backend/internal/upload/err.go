package upload

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrParentNotDirectory = errors.New("parent is not a directory")
	ErrConflict           = errors.New("node conflict")
)
