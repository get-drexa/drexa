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

type LoginResult struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	userService *user.Service
	tokenConfig TokenConfig
}

type registerOptions struct {
	displayName string
	email       string
	password    string
}

func NewService(userService *user.Service, tokenConfig TokenConfig) *Service {
	return &Service{
		userService: userService,
		tokenConfig: tokenConfig,
	}
}

func (s *Service) LoginWithEmailAndPassword(ctx context.Context, db bun.IDB, email, plain string) (*LoginResult, error) {
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

	return &LoginResult{
		User:         u,
		AccessToken:  at,
		RefreshToken: hex.EncodeToString(rt.Token),
	}, nil
}

func (s *Service) Register(ctx context.Context, db bun.IDB, opts registerOptions) (*LoginResult, error) {
	hashed, err := password.Hash(opts.password)
	if err != nil {
		return nil, err
	}

	u, err := s.userService.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       opts.email,
		DisplayName: opts.displayName,
		Password:    hashed,
	})
	if err != nil {
		return nil, err
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

	return &LoginResult{
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
