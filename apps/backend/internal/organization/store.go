package organization

import (
	"context"

	"github.com/google/uuid"
)

// Store provides persistence operations for organization records.
type Store interface {
	CreatePersonal(ctx context.Context, name string) (*Organization, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	FindPersonalForUser(ctx context.Context, userID uuid.UUID) (*Organization, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]Organization, error)
}
