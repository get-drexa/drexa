package user

import (
	"fmt"

	"github.com/google/uuid"
)

type NotFoundError struct {
	// ID is the ID that was used to try to find the user.
	// Not set if not tried.
	id uuid.UUID

	// Email is the email that was used to try to find the user.
	// Not set if not tried.
	email string
}

func newNotFoundError(id uuid.UUID, email string) *NotFoundError {
	return &NotFoundError{id, email}
}
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("user not found: %v", e.id)
}

type AlreadyExistsError struct {
	// Email is the email that was used to try to create the user.
	Email string
}

func newAlreadyExistsError(email string) *AlreadyExistsError {
	return &AlreadyExistsError{email}
}
func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("user with email %s already exists", e.Email)
}
