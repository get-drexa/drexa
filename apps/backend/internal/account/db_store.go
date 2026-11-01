package account

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

func (s *dbStore) Create(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role Role, status Status) (*Account, error) {
	id, err := newAccountID()
	if err != nil {
		return nil, err
	}

	acc := &Account{
		ID:     id,
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
		Status: status,
	}

	_, err = s.db.NewInsert().Model(acc).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrAccountAlreadyExists
		}

		return nil, err
	}

	return acc, nil
}

func (s *dbStore) ListForUser(ctx context.Context, userID uuid.UUID) ([]*Account, error) {
	var accounts []*Account

	err := s.db.NewSelect().Model(&accounts).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Account, 0), nil
		}

		return nil, err
	}

	return accounts, nil
}

func (s *dbStore) FindByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*Account, error) {
	var acc Account

	err := s.db.NewSelect().Model(&acc).Where("user_id = ?", userID).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}

	return &acc, nil
}

func (s *dbStore) FindForUserInOrg(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*Account, error) {
	var acc Account

	err := s.db.NewSelect().Model(&acc).
		Where("org_id = ?", orgID).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}

	return &acc, nil
}
