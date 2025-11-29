package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct{}

type UserRegistrationOptions struct {
	Email       string
	DisplayName string
	Password    password.Hashed
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) RegisterUser(ctx context.Context, db bun.IDB, opts UserRegistrationOptions) (*User, error) {
	uid, err := newUserID()
	if err != nil {
		return nil, err
	}

	u := User{
		ID:          uid,
		Email:       opts.Email,
		DisplayName: opts.DisplayName,
		Password:    opts.Password,
	}

	_, err = db.NewInsert().Model(&u).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, newAlreadyExistsError(u.Email)
		}
		return nil, err
	}

	return &u, nil
}

func (s *Service) UserByID(ctx context.Context, db bun.IDB, id uuid.UUID) (*User, error) {
	var user User
	err := db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newNotFoundError(id, "")
		}
		return nil, err
	}
	return &user, nil
}

func (s *Service) UserByEmail(ctx context.Context, db bun.IDB, email string) (*User, error) {
	var user User
	err := db.NewSelect().Model(&user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newNotFoundError(uuid.Nil, email)
		}
		return nil, err
	}
	return &user, nil
}

func (s *Service) UserExistsByEmail(ctx context.Context, db bun.IDB, email string) (bool, error) {
	return db.NewSelect().Model(&User{}).Where("email = ?", email).Exists(ctx)
}
