package drive

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Drive struct {
	bun.BaseModel `bun:"drives" swaggerignore:"true"`

	ID        uuid.UUID `bun:",pk,type:uuid" json:"id"`
	PublicID  string    `bun:"public_id,notnull" json:"publicId"`
	OrgID     uuid.UUID `bun:"org_id,notnull,type:uuid" json:"orgId"`
	Name      string    `bun:"name,notnull" json:"name"`

	OwnerAccountID *uuid.UUID `bun:"owner_account_id,type:uuid" json:"ownerAccountId,omitempty"`

	StorageUsageBytes int64 `bun:"storage_usage_bytes,notnull" json:"storageUsageBytes"`
	StorageQuotaBytes int64 `bun:"storage_quota_bytes,notnull" json:"storageQuotaBytes"`

	CreatedAt time.Time `bun:"created_at,notnull,nullzero" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero" json:"updatedAt"`
}

func newDriveID() (uuid.UUID, error) {
	return uuid.NewV7()
}
