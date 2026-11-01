package user

import (
	"context"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/google/uuid"
)

type Service interface {
	RegisterUser(ctx context.Context, opts UserRegistrationOptions) (*User, error)
	UserByID(ctx context.Context, id uuid.UUID) (*User, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
}

type service struct {
	store Store
}

var _ Service = (*service)(nil)

type UserRegistrationOptions struct {
	Email       string
	DisplayName string
	Password    password.Hashed
}

func NewService(store Store) Service {
	return &service{store: store}
}

func (s *service) RegisterUser(ctx context.Context, opts UserRegistrationOptions) (*User, error) {
	return s.store.Create(ctx, CreateOptions(opts))
}

func (s *service) UserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.store.FindByID(ctx, id)
}

func (s *service) UserByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.FindByEmail(ctx, email)
}

func (s *service) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	return s.store.ExistsByEmail(ctx, email)
}
