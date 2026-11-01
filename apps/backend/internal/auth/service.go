package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuthenticationTokens struct {
	Organizations []organization.Organization
	User          *user.User
	AccessToken   string
	RefreshToken  string
}

type AccessTokenAuthentication struct {
	Claims *jwt.RegisteredClaims
	User   *user.User
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service interface {
	LoginWithEmailAndPassword(ctx context.Context, email, plain string) (*AuthenticationTokens, error)
	GrantForUser(ctx context.Context, db bun.IDB, user *user.User) (*AuthenticationTokens, error)
	AuthenticateWithAccessToken(ctx context.Context, token string) (*AccessTokenAuthentication, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthenticationTokens, error)
}

type service struct {
	db          *bun.DB
	users       user.Store
	tokenConfig TokenConfig
}

var _ Service = (*service)(nil)

func NewService(db *bun.DB, users user.Store, tokenConfig TokenConfig) Service {
	return &service{
		db:          db,
		users:       users,
		tokenConfig: tokenConfig,
	}
}

func (s *service) LoginWithEmailAndPassword(ctx context.Context, email, plain string) (*AuthenticationTokens, error) {
	var result *AuthenticationTokens

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		u, err := s.authenticateWithEmailAndPassword(ctx, tx, email, plain)
		if err != nil {
			return err
		}

		result, err = s.GrantForUser(ctx, tx, u)
		if err != nil {
			return err
		}

		result.Organizations, err = organization.NewDBStore(tx).ListForUser(ctx, u.ID)

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GrantForUser joins the caller's transaction, including during registration.
func (s *service) GrantForUser(ctx context.Context, db bun.IDB, user *user.User) (*AuthenticationTokens, error) {
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

func (s *service) AuthenticateWithAccessToken(ctx context.Context, token string) (*AccessTokenAuthentication, error) {
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

	u, err := s.users.FindByID(ctx, id)
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

func (s *service) refreshAccessToken(ctx context.Context, db bun.IDB, refreshToken string) (*AuthenticationTokens, error) {
	t, err := base64.URLEncoding.DecodeString(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	rtParts, err := DeserializeRefreshToken(t)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	rt := &RefreshToken{}
	// Hold the row lock until commit so only one request can create a successor.
	err = db.NewSelect().Model(rt).Where("key = ?", rtParts.Key).For("UPDATE").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}

		return nil, err
	}

	ok, err := password.Verify(rtParts.Token, rt.TokenHash)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrInvalidRefreshToken
	}

	now := time.Now()
	if rt.ExpiresAt.Before(now) {
		return nil, ErrRefreshTokenExpired
	}

	if rt.ConsumedAt != nil {
		// Evaluate after acquiring the lock. Retries never move ConsumedAt.
		if now.Before(rt.ConsumedAt.Add(refreshTokenReuseGrace)) {
			return nil, ErrRefreshTokenRotated
		}

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

	u, err := user.NewDBStore(db).FindByID(ctx, grant.UserID)
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

func (s *service) generateTokens(ctx context.Context, db bun.IDB, user *user.User, grant *Grant) (*AuthenticationTokens, error) {
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

func (s *service) authenticateWithEmailAndPassword(ctx context.Context, db bun.IDB, email, plain string) (*user.User, error) {
	u, err := user.NewDBStore(db).FindByEmail(ctx, email)
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

// RefreshAccessToken commits revocation/cleanup for rejected tokens as well as
// successful rotations. Infrastructure failures roll back the transaction.
func (s *service) RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthenticationTokens, error) {
	var result *AuthenticationTokens
	var tokenErr error

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, tokenErr = s.refreshAccessToken(ctx, tx, refreshToken)
		if errors.Is(tokenErr, ErrInvalidRefreshToken) ||
			errors.Is(tokenErr, ErrRefreshTokenExpired) ||
			errors.Is(tokenErr, ErrRefreshTokenReused) ||
			errors.Is(tokenErr, ErrRefreshTokenRotated) {
			return nil
		}

		return tokenErr
	})
	if err != nil {
		return nil, err
	}

	if tokenErr != nil {
		return nil, tokenErr
	}

	return result, nil
}
