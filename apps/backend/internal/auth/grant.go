package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Grant struct {
	bun.BaseModel `bun:"grants"`

	ID        uuid.UUID  `bun:",pk,type:uuid"`
	UserID    uuid.UUID  `bun:"user_id,notnull"`
	CreatedAt time.Time  `bun:"created_at,notnull,nullzero"`
	UpdatedAt time.Time  `bun:"updated_at,notnull,nullzero"`
	RevokedAt *time.Time `bun:"revoked_at,nullzero"`
}

func newGrantID() (uuid.UUID, error) {
	return uuid.NewV7()
}
