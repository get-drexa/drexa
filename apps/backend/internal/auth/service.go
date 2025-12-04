package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuthenticationTokens struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
}

type AccessTokenAuthentication struct {
	Claims *jwt.RegisteredClaims
	User   *user.User
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

func (s *Service) AuthenticateWithAccessToken(ctx context.Context, db bun.IDB, token string) (*AccessTokenAuthentication, error) {
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

	u, err := s.userService.UserByID(ctx, db, id)
	if err != nil {
		var nf *user.NotFoundError
		if errors.As(err, &nf) {
			return nil, newInvalidAccessTokenError(err)
		}
		return nil, err
	}

	return &AccessTokenAuthentication{
		Claims: claims,
		User:   u,
	}, nil
}

func (s *Service) RefreshAccessToken(ctx context.Context, db bun.IDB, refreshToken string) (*AuthenticationTokens, error) {
	t, err := base64.URLEncoding.DecodeString(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	rtParts, err := DeserializeRefreshToken(t)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	rt := &RefreshToken{}
	err = db.NewSelect().Model(rt).Where("key = ?", rtParts.Key).Scan(ctx)
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

	ok, err := password.Verify(rtParts.Token, rt.TokenHash)
	if err != nil {
		return nil, err
	}
	if !ok {
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

	srt, err := SerializeRefreshToken(newRT)
	if err != nil {
		return nil, err
	}

	return &AuthenticationTokens{
		User:         u,
		AccessToken:  at,
		RefreshToken: base64.URLEncoding.EncodeToString(srt),
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

	srt, err := SerializeRefreshToken(rt)
	if err != nil {
		return nil, err
	}

	return &AuthenticationTokens{
		User:         user,
		AccessToken:  at,
		RefreshToken: base64.URLEncoding.EncodeToString(srt),
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

	ok, err := password.VerifyString(plain, u.Password)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}
