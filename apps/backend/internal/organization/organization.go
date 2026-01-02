package organization

import (
	"encoding/json"
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
	Slug *string   `bun:"slug" json:"slug,omitempty" example:"test-org"`

	CreatedAt time.Time `bun:"created_at,notnull,nullzero" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero" json:"updatedAt"`
}

func (o Organization) MarshalJSON() ([]byte, error) {
	type organizationJSON struct {
		ID        uuid.UUID `json:"id"`
		Kind      Kind      `json:"kind"`
		Name      string    `json:"name"`
		Slug      *string   `json:"slug,omitempty"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	slug := o.Slug
	if o.Kind == KindPersonal {
		s := ReservedSlug
		slug = &s
	}

	return json.Marshal(organizationJSON{
		ID:        o.ID,
		Kind:      o.Kind,
		Name:      o.Name,
		Slug:      slug,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	})
}

func newOrganizationID() (uuid.UUID, error) {
	return uuid.NewV7()
}
