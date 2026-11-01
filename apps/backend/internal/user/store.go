package user

import (
	"context"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/google/uuid"
)

type CreateOptions struct {
	Email       string
	DisplayName string
	Password    password.Hashed
}

type Store interface {
	Create(ctx context.Context, opts CreateOptions) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
