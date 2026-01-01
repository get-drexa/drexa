package account

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Status string

const (
	StatusInvited   Status = "invited"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)

// Account represents a user's identity within an organization (principal / membership).
type Account struct {
	bun.BaseModel `bun:"accounts" swaggerignore:"true"`

	ID     uuid.UUID `bun:",pk,type:uuid" json:"id"`
	OrgID  uuid.UUID `bun:"org_id,notnull,type:uuid" json:"orgId"`
	UserID uuid.UUID `bun:"user_id,notnull,type:uuid" json:"userId"`

	Role   Role   `bun:"role,notnull" json:"role" example:"member"`
	Status Status `bun:"status,notnull" json:"status" example:"active"`

	CreatedAt time.Time `bun:"created_at,notnull,nullzero" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero" json:"updatedAt"`
}

func newAccountID() (uuid.UUID, error) {
	return uuid.NewV7()
}

