package account

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	CreateAccount(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role Role, status Status) (*Account, error)
	ListAccounts(ctx context.Context, userID uuid.UUID) ([]*Account, error)
	AccountByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*Account, error)
	FindUserAccountInOrg(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*Account, error)
}

type service struct {
	store Store
}

var _ Service = (*service)(nil)

func NewService(store Store) Service {
	return &service{store: store}
}

func (s *service) CreateAccount(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role Role, status Status) (*Account, error) {
	return s.store.Create(ctx, orgID, userID, role, status)
}

func (s *service) ListAccounts(ctx context.Context, userID uuid.UUID) ([]*Account, error) {
	return s.store.ListForUser(ctx, userID)
}

func (s *service) AccountByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*Account, error) {
	return s.store.FindByID(ctx, userID, id)
}

func (s *service) FindUserAccountInOrg(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*Account, error) {
	return s.store.FindForUserInOrg(ctx, orgID, userID)
}
