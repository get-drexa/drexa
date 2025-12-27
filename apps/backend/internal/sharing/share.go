package sharing

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Share represents a share link for files or directories
// @Description Share link information including expiration and timestamps
type Share struct {
	bun.BaseModel `bun:"node_shares"`

	ID        uuid.UUID `bun:",pk,type:uuid" json:"-"`
	AccountID uuid.UUID `bun:"account_id,notnull,type:uuid" json:"-"`
	// Unique share identifier (public ID)
	PublicID          string     `bun:"public_id,notnull" json:"id" example:"kRp2XYTq9A55"`
	SharedDirectoryID uuid.UUID  `bun:"shared_directory_id,notnull,type:uuid" json:"-"`
	RevokedAt         *time.Time `bun:"revoked_at" json:"-"`
	// When the share expires, null if it never expires (ISO 8601)
	ExpiresAt *time.Time `bun:"expires_at" json:"expiresAt" example:"2025-01-15T00:00:00Z"`
	// When the share was created (ISO 8601)
	CreatedAt time.Time `bun:"created_at,notnull" json:"createdAt" example:"2024-12-13T15:04:05Z"`
	// When the share was last updated (ISO 8601)
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updatedAt" example:"2024-12-13T16:30:00Z"`
}

type SharePermission struct {
	bun.BaseModel `bun:"share_permissions"`

	ID        uuid.UUID  `bun:",pk,type:uuid"`
	ShareID   uuid.UUID  `bun:"share_id,notnull,type:uuid"`
	AccountID *uuid.UUID `bun:"account_id,type:uuid"`
	CanRead   bool       `bun:"can_read,notnull"`
	CanWrite  bool       `bun:"can_write,notnull"`
	CanDelete bool       `bun:"can_delete,notnull"`
	CanUpload bool       `bun:"can_upload,notnull"`
	ExpiresAt *time.Time `bun:"expires_at"`
	CreatedAt time.Time  `bun:"created_at,notnull"`
	UpdatedAt time.Time  `bun:"updated_at,notnull"`
}

type ShareItem struct {
	bun.BaseModel `bun:"share_items"`

	ID        uuid.UUID `bun:",pk,type:uuid"`
	ShareID   uuid.UUID `bun:"share_id,notnull,type:uuid"`
	NodeID    uuid.UUID `bun:"node_id,notnull,type:uuid"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
}

func generateInternalID() (uuid.UUID, error) {
	return uuid.NewV7()
}
