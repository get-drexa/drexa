package drive

import "errors"

var (
	ErrDriveNotFound   = errors.New("drive not found")
	ErrDriveForbidden  = errors.New("drive forbidden")
	ErrDriveNotAllowed = errors.New("drive not allowed")
)
