package blob

import "errors"

var (
	ErrConflict           = errors.New("key already used for a different blob")
	ErrNotFound           = errors.New("key not found")
	ErrInvalidFileContent = errors.New("invalid file content. must provide either a reader or a blob key")
)
