package account

import (
	"context"

	"github.com/google/uuid"
)

// Store provides persistence operations for account records.
type Store interface {
	Create(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role Role, status Status) (*Account, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*Account, error)
	FindByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*Account, error)
	FindForUserInOrg(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*Account, error)
}
