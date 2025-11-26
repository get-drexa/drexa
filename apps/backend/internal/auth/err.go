package auth

import (
	"errors"
	"fmt"
)

var ErrUnauthenticatedRequest = errors.New("unauthenticated request")

type InvalidAccessTokenError struct {
	err error
}

func newInvalidAccessTokenError(err error) *InvalidAccessTokenError {
	return &InvalidAccessTokenError{err}
}

func (e *InvalidAccessTokenError) Error() string {
	return fmt.Sprintf("invalid access token: %v", e.err)
}

func (e *InvalidAccessTokenError) Unwrap() error {
	return e.err
}
