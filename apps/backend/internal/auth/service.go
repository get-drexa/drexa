package auth

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuthenticationResult struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	userService *user.Service
	tokenConfig TokenConfig
}

func NewService(userService *user.Service, tokenConfig TokenConfig) *Service {
	return &Service{
		userService: userService,
		tokenConfig: tokenConfig,
	}
}

func (s *Service) GenerateTokenForUser(ctx context.Context, db bun.IDB, user *user.User) (*AuthenticationResult, error) {
	at, err := GenerateAccessToken(user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	_, err = db.NewInsert().Model(rt).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &AuthenticationResult{
		User:         user,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}

func (s *Service) AuthenticateWithEmailAndPassword(ctx context.Context, db bun.IDB, email, plain string) (*AuthenticationResult, error) {
	u, err := s.userService.UserByEmail(ctx, db, email)
	if err != nil {
		var nf *user.NotFoundError
		if errors.As(err, &nf) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	ok, err := password.Verify(plain, u.Password)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	at, err := GenerateAccessToken(u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	_, err = db.NewInsert().Model(rt).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &AuthenticationResult{
		User:         u,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}

func (s *Service) AuthenticateWithAccessToken(ctx context.Context, db bun.IDB, token string) (*user.User, error) {
	claims, err := ParseAccessToken(token, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, newInvalidAccessTokenError(err)
	}

	return s.userService.UserByID(ctx, db, id)
}
