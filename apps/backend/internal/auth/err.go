package auth

import (
	"errors"
	"fmt"
)

var ErrInvalidRefreshToken = errors.New("invalid refresh token")
var ErrRefreshTokenExpired = errors.New("refresh token expired")
var ErrRefreshTokenReused = errors.New("refresh token reused")

// ErrRefreshTokenRotated means another request consumed this token within grace.
// It grants no credentials; callers must retry with the latest token.
var ErrRefreshTokenRotated = errors.New("refresh token already rotated")

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
