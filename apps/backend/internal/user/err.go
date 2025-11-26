package user

import (
	"fmt"

	"github.com/google/uuid"
)

type NotFoundError struct {
	id uuid.UUID
}

func newNotFoundError(id uuid.UUID) *NotFoundError {
	return &NotFoundError{id}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("user not found: %v", e.id)
}
