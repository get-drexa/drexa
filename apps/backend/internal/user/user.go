package user

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"users"`

	ID                uuid.UUID `bun:",pk,type:uuid" json:"id"`
	DisplayName       string    `bun:"display_name,notnull" json:"displayName"`
	Email             string    `bun:"email,unique,notnull" json:"email"`
	Password          string    `bun:"password,notnull" json:"-"`
	StorageUsageBytes int64     `bun:"storage_usage_bytes,notnull" json:"storageUsageBytes"`
	StorageQuotaBytes int64     `bun:"storage_quota_bytes,notnull" json:"storageQuotaBytes"`
}
