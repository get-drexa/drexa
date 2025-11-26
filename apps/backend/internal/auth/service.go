package auth

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/get-drexa/drexa/internal/user"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LoginResult struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserExists = errors.New("user already exists")

type Service struct {
	db          *bun.DB
	userService *user.Service
	tokenConfig TokenConfig
}

type registerOptions struct {
	displayName string
	email       string
	password    string
}

func NewService(db *bun.DB, userService *user.Service, tokenConfig TokenConfig) *Service {
	return &Service{
		db:          db,
		userService: userService,
		tokenConfig: tokenConfig,
	}
}

func (s *Service) LoginWithEmailAndPassword(ctx context.Context, email, password string) (*LoginResult, error) {
	var u user.User

	err := s.db.NewSelect().Model(&u).Where("email = ?", email).Scan(ctx)
	if err != nil {
		return nil, err
	}

	ok, err := VerifyPassword(password, u.Password)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	at, err := GenerateAccessToken(&u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(&u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	_, err = s.db.NewInsert().Model(rt).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         &u,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}

func (s *Service) Register(ctx context.Context, opts registerOptions) (*LoginResult, error) {
	u := user.User{
		Email:       opts.email,
		DisplayName: opts.displayName,
	}

	exists, err := s.db.NewSelect().Model(&u).Where("email = ?", opts.email).Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	u.Password, err = HashPassword(opts.password)
	if err != nil {
		return nil, err
	}

	_, err = s.db.NewInsert().Model(&u).Exec(ctx)
	if err != nil {
		return nil, err
	}

	at, err := GenerateAccessToken(&u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(&u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         &u,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}

func (s *Service) AuthenticateWithAccessToken(ctx context.Context, token string) (*user.User, error) {
	claims, err := ParseAccessToken(token, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, newInvalidAccessTokenError(err)
	}

	return s.userService.UserByID(ctx, id)
}
