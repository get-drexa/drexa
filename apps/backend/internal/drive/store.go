package drive

import (
	"context"

	"github.com/google/uuid"
)

// Store provides persistence operations for drive records.
type Store interface {
	Create(ctx context.Context, opts CreateDriveOptions) (*Drive, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Drive, error)
	FindByPublicID(ctx context.Context, publicID string) (*Drive, error)
	ListAccessible(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) ([]*Drive, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*Drive, error)
}
