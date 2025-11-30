package user

import (
	"time"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"users"`

	ID          uuid.UUID       `bun:",pk,type:uuid" json:"id"`
	DisplayName string          `bun:"display_name" json:"displayName"`
	Email       string          `bun:"email,unique,notnull" json:"email"`
	Password    password.Hashed `bun:"password,notnull" json:"-"`
	CreatedAt   time.Time       `bun:"created_at,notnull,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,notnull,nullzero" json:"updatedAt"`
}

func newUserID() (uuid.UUID, error) {
	return uuid.NewV7()
}
