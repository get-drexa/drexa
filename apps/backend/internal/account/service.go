package account

import (
	"context"
	"database/sql"
	"errors"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CreateAccount(ctx context.Context, db bun.IDB, orgID uuid.UUID, userID uuid.UUID, role Role, status Status) (*Account, error) {
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

	_, err = db.NewInsert().Model(acc).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrAccountAlreadyExists
		}
		return nil, err
	}
	return acc, nil
}

func (s *Service) ListAccounts(ctx context.Context, db bun.IDB, userID uuid.UUID) ([]*Account, error) {
	var accounts []*Account
	err := db.NewSelect().Model(&accounts).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Account, 0), nil
		}
		return nil, err
	}
	return accounts, nil
}

func (s *Service) AccountByID(ctx context.Context, db bun.IDB, userID uuid.UUID, id uuid.UUID) (*Account, error) {
	var acc Account
	err := db.NewSelect().Model(&acc).Where("user_id = ?", userID).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &acc, nil
}

func (s *Service) FindUserAccountInOrg(ctx context.Context, db bun.IDB, orgID uuid.UUID, userID uuid.UUID) (*Account, error) {
	var acc Account
	err := db.NewSelect().Model(&acc).
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
