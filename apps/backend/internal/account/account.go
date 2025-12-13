package account

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Account represents a storage account with quota information
// @Description Storage account with usage and quota details
type Account struct {
	bun.BaseModel `bun:"accounts" swaggerignore:"true"`

	// Unique account identifier
	ID uuid.UUID `bun:",pk,type:uuid" json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// ID of the user who owns this account
	UserID uuid.UUID `bun:"user_id,notnull,type:uuid" json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	// Current storage usage in bytes
	StorageUsageBytes int64 `bun:"storage_usage_bytes,notnull" json:"storageUsageBytes" example:"1073741824"`
	// Maximum storage quota in bytes
	StorageQuotaBytes int64 `bun:"storage_quota_bytes,notnull" json:"storageQuotaBytes" example:"10737418240"`
	// When the account was created (ISO 8601)
	CreatedAt time.Time `bun:"created_at,notnull,nullzero" json:"createdAt" example:"2024-12-13T15:04:05Z"`
	// When the account was last updated (ISO 8601)
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero" json:"updatedAt" example:"2024-12-13T16:30:00Z"`
}

func newAccountID() (uuid.UUID, error) {
	return uuid.NewV7()
}
