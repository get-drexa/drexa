package auth

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/uptrace/bun"
)

type LoginResult struct {
	User         User
	AccessToken  string
	RefreshToken string
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserExists = errors.New("user already exists")

type Service struct {
	db          *bun.DB
	tokenConfig TokenConfig
}

type registerOptions struct {
	displayName string
	email       string
	password    string
}

func NewService(db *bun.DB, tokenConfig TokenConfig) *Service {
	return &Service{
		db:          db,
		tokenConfig: tokenConfig,
	}
}

func (s *Service) LoginWithEmailAndPassword(ctx context.Context, email, password string) (*LoginResult, error) {
	var user User

	err := s.db.NewSelect().Model(&user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		return nil, err
	}

	ok, err := VerifyPassword(password, user.Password)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	at, err := GenerateAccessToken(&user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(&user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	_, err = s.db.NewInsert().Model(rt).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         user,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}

func (s *Service) Register(ctx context.Context, opts registerOptions) (*LoginResult, error) {
	user := User{
		Email:       opts.email,
		DisplayName: opts.displayName,
	}

	exists, err := s.db.NewSelect().Model(&user).Where("email = ?", opts.email).Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	user.Password, err = HashPassword(opts.password)
	if err != nil {
		return nil, err
	}

	_, err = s.db.NewInsert().Model(&user).Exec(ctx)
	if err != nil {
		return nil, err
	}

	at, err := GenerateAccessToken(&user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(&user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         user,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}
