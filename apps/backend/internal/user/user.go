package user

import (
	"time"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User represents a user account in the system
// @Description User account information
type User struct {
	bun.BaseModel `bun:"users" swaggerignore:"true"`

	// Unique user identifier
	ID uuid.UUID `bun:",pk,type:uuid" json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// User's display name
	DisplayName string `bun:"display_name" json:"displayName" example:"John Doe"`
	// User's email address
	Email       string          `bun:"email,unique,notnull" json:"email" example:"john@example.com"`
	Password    password.Hashed `bun:"password,notnull" json:"-" swaggerignore:"true"`
	CreatedAt   time.Time       `bun:"created_at,notnull,nullzero" json:"-" swaggerignore:"true"`
	UpdatedAt   time.Time       `bun:"updated_at,notnull,nullzero" json:"-" swaggerignore:"true"`
}

func newUserID() (uuid.UUID, error) {
	return uuid.NewV7()
}
