package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuthenticationTokens struct {
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

func (s *Service) LoginWithEmailAndPassword(ctx context.Context, db bun.IDB, email, plain string) (*AuthenticationTokens, error) {
	u, err := s.authenticateWithEmailAndPassword(ctx, db, email, plain)
	if err != nil {
		return nil, err
	}
	return s.GrantForUser(ctx, db, u)
}

func (s *Service) GrantForUser(ctx context.Context, db bun.IDB, user *user.User) (*AuthenticationTokens, error) {
	id, err := newGrantID()
	if err != nil {
		return nil, err
	}

	grant := &Grant{
		ID:     id,
		UserID: user.ID,
	}
	_, err = db.NewInsert().Model(grant).Exec(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.generateTokens(ctx, db, user, grant)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) AuthenticateWithAccessToken(ctx context.Context, db bun.IDB, token string) (*user.User, error) {
	claims, err := ParseAccessToken(token, &s.tokenConfig)
	if err != nil {
		slog.Info("failed to parse access token", "error", err)
		return nil, err
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		slog.Info("failed to parse access token subject", "error", err)
		return nil, newInvalidAccessTokenError(err)
	}

	return s.userService.UserByID(ctx, db, id)
}

func (s *Service) RefreshAccessToken(ctx context.Context, db bun.IDB, refreshToken string) (*AuthenticationTokens, error) {
	rtBytes, err := DecodeRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	rtHash := HashRefreshToken(rtBytes)

	rt := &RefreshToken{}
	err = db.NewSelect().Model(rt).Where("token_hash = ?", rtHash).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if rt.ExpiresAt.Before(time.Now()) {
		return nil, ErrRefreshTokenExpired
	}

	if rt.ConsumedAt != nil {
		// token reuse detected, invalidate all refresh tokens under the same grant
		_, err = db.NewDelete().Model((*RefreshToken)(nil)).Where("grant_id = ?", rt.GrantID).Exec(ctx)
		if err != nil {
			return nil, err
		}

		_, err = db.NewUpdate().Model((*Grant)(nil)).Set("revoked_at = ?", time.Now()).Where("id = ?", rt.GrantID).Exec(ctx)
		if err != nil {
			return nil, err
		}

		return nil, ErrRefreshTokenReused
	}

	grant := &Grant{
		ID: rt.GrantID,
	}
	err = db.NewSelect().Model(grant).WherePK().Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// the grant for this refresh token was deleted, delete the refresh token
			_, _ = db.NewDelete().Model(rt).WherePK().Exec(ctx)
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if grant.RevokedAt != nil {
		return nil, ErrInvalidRefreshToken
	}

	u, err := s.userService.UserByID(ctx, db, grant.UserID)
	if err != nil {
		var nf *user.NotFoundError
		if errors.As(err, &nf) {
			// the user for this grant was deleted, delete the grant and refresh token
			_, _ = db.NewDelete().Model(grant).WherePK().Exec(ctx)
			_, _ = db.NewDelete().Model(rt).WherePK().Exec(ctx)
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	newRT, err := GenerateRefreshToken(u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}
	newRT.GrantID = grant.ID

	_, err = db.NewUpdate().Model(rt).Set("consumed_at = ?", time.Now()).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	_, err = db.NewInsert().Model(newRT).Exec(ctx)
	if err != nil {
		return nil, err
	}

	at, err := GenerateAccessToken(u, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	return &AuthenticationTokens{
		User:         u,
		AccessToken:  at,
		RefreshToken: EncodeRefreshToken(newRT.Token),
	}, nil
}

func (s *Service) generateTokens(ctx context.Context, db bun.IDB, user *user.User, grant *Grant) (*AuthenticationTokens, error) {
	at, err := GenerateAccessToken(user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt, err := GenerateRefreshToken(user, &s.tokenConfig)
	if err != nil {
		return nil, err
	}

	rt.GrantID = grant.ID

	_, err = db.NewInsert().Model(rt).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &AuthenticationTokens{
		User:         user,
		AccessToken:  at,
		RefreshToken: EncodeRefreshToken(rt.Token),
	}, nil
}

func (s *Service) authenticateWithEmailAndPassword(ctx context.Context, db bun.IDB, email, plain string) (*user.User, error) {
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

	return u, nil
}
