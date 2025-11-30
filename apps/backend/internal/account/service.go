package account

import (
	"context"
	"database/sql"
	"errors"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct {
	userService user.Service
}

type RegisterOptions struct {
	Email       string
	Password    string
	DisplayName string
}

type CreateAccountOptions struct {
	OrganizationID uuid.UUID
	QuotaBytes     int64
}

func NewService(userService *user.Service) *Service {
	return &Service{
		userService: *userService,
	}
}

func (s *Service) Register(ctx context.Context, db bun.IDB, opts RegisterOptions) (*Account, *user.User, error) {
	hashed, err := password.Hash(opts.Password)
	if err != nil {
		return nil, nil, err
	}

	u, err := s.userService.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       opts.Email,
		Password:    hashed,
		DisplayName: opts.DisplayName,
	})
	if err != nil {
		return nil, nil, err
	}

	acc, err := s.CreateAccount(ctx, db, u.ID, CreateAccountOptions{
		// TODO: make quota configurable
		QuotaBytes: 1024 * 1024 * 1024, // 1GB
	})
	if err != nil {
		return nil, nil, err
	}

	return acc, u, nil
}

func (s *Service) CreateAccount(ctx context.Context, db bun.IDB, userID uuid.UUID, opts CreateAccountOptions) (*Account, error) {
	id, err := newAccountID()
	if err != nil {
		return nil, err
	}

	account := &Account{
		ID:                id,
		UserID:            userID,
		StorageQuotaBytes: opts.QuotaBytes,
	}

	_, err = db.NewInsert().Model(account).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrAccountAlreadyExists
		}
		return nil, err
	}

	return account, nil
}

func (s *Service) AccountByUserID(ctx context.Context, db bun.IDB, userID uuid.UUID) (*Account, error) {
	var account Account
	err := db.NewSelect().Model(&account).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (s *Service) AccountByID(ctx context.Context, db bun.IDB, userID uuid.UUID, id uuid.UUID) (*Account, error) {
	var account Account
	err := db.NewSelect().Model(&account).Where("user_id = ?", userID).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}
