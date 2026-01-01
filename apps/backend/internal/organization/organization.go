package organization

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Kind string

const (
	KindPersonal Kind = "personal"
	KindTeam     Kind = "team"
)

type Organization struct {
	bun.BaseModel `bun:"organizations" swaggerignore:"true"`

	ID   uuid.UUID `bun:",pk,type:uuid" json:"id"`
	Kind Kind      `bun:"kind,notnull" json:"kind" example:"personal"`
	Name string    `bun:"name,notnull" json:"name" example:"Personal"`

	CreatedAt time.Time `bun:"created_at,notnull,nullzero" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero" json:"updatedAt"`
}

func newOrganizationID() (uuid.UUID, error) {
	return uuid.NewV7()
}

