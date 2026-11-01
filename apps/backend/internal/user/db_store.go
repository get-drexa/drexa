package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type dbStore struct {
	db bun.IDB
}

var _ Store = (*dbStore)(nil)

// NewDBStore uses db for every query. Pass a bun.Tx to participate in an
// existing transaction; the caller is responsible for committing or rolling back.
func NewDBStore(db bun.IDB) *dbStore {
	return &dbStore{db: db}
}

func (s *dbStore) Create(ctx context.Context, opts CreateOptions) (*User, error) {
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

	_, err = s.db.NewInsert().Model(&u).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, newAlreadyExistsError(u.Email)
		}

		return nil, err
	}

	return &u, nil
}

func (s *dbStore) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User

	err := s.db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newNotFoundError(id, "")
		}

		return nil, err
	}

	return &user, nil
}

func (s *dbStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	err := s.db.NewSelect().Model(&user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newNotFoundError(uuid.Nil, email)
		}

		return nil, err
	}

	return &user, nil
}

func (s *dbStore) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return s.db.NewSelect().Model(&User{}).Where("email = ?", email).Exists(ctx)
}
