package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/get-drexa/drexa/internal/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	accessTokenValidFor    = time.Minute * 15
	refreshTokenByteLength = 32
	refreshTokenValidFor   = time.Hour * 24 * 30
)

type TokenConfig struct {
	Issuer    string
	Audience  string
	SecretKey []byte
}

type RefreshToken struct {
	bun.BaseModel `bun:"refresh_tokens"`

	ID        uuid.UUID `bun:",pk,type:uuid"`
	UserID    uuid.UUID `bun:"user_id,notnull"`
	Token     []byte    `bun:"-"`
	TokenHash string    `bun:"token_hash,notnull"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull"`
}

func newTokenID() (uuid.UUID, error) {
	return uuid.NewV7()
}

func GenerateAccessToken(user *user.User, c *TokenConfig) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    c.Issuer,
		Audience:  jwt.ClaimStrings{c.Audience},
		Subject:   user.ID.String(),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenValidFor)),
		IssuedAt:  jwt.NewNumericDate(now),
	})

	signed, err := token.SignedString(c.SecretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

func GenerateRefreshToken(user *user.User, c *TokenConfig) (*RefreshToken, error) {
	now := time.Now()

	buf := make([]byte, refreshTokenByteLength)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	id, err := newTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token ID: %w", err)
	}

	h := sha256.Sum256(buf)
	hex := hex.EncodeToString(h[:])

	return &RefreshToken{
		ID:        id,
		UserID:    user.ID,
		Token:     buf,
		TokenHash: hex,
		ExpiresAt: now.Add(refreshTokenValidFor),
		CreatedAt: now,
	}, nil
}

// ParseAccessToken parses a JWT access token and returns the claims.
// Returns an InvalidAccessTokenError if the token is invalid.
func ParseAccessToken(token string, c *TokenConfig) (*jwt.RegisteredClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return c.SecretKey, nil
	}, jwt.WithIssuer(c.Issuer), jwt.WithExpirationRequired(), jwt.WithAudience(c.Audience))
	if err != nil {
		return nil, newInvalidAccessTokenError(err)
	}
	return parsed.Claims.(*jwt.RegisteredClaims), nil
}
