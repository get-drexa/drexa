package account

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `bun:"accounts"`

	ID                uuid.UUID `bun:",pk,type:uuid" json:"id"`
	UserID            uuid.UUID `bun:"user_id,notnull,type:uuid" json:"userId"`
	StorageUsageBytes int64     `bun:"storage_usage_bytes,notnull" json:"storageUsageBytes"`
	StorageQuotaBytes int64     `bun:"storage_quota_bytes,notnull" json:"storageQuotaBytes"`
	CreatedAt         time.Time `bun:"created_at,notnull,nullzero" json:"createdAt"`
	UpdatedAt         time.Time `bun:"updated_at,notnull,nullzero" json:"updatedAt"`
}

func newAccountID() (uuid.UUID, error) {
	return uuid.NewV7()
}
