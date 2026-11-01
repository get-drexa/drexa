package auth

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	accessTokenValidFor    = time.Minute * 15
	refreshTokenKeyLength  = 16
	refreshTokenDataLength = 32
	refreshTokenLength     = refreshTokenKeyLength + refreshTokenDataLength
	refreshTokenValidFor   = time.Hour * 24 * 30
	refreshTokenReuseGrace = 10 * time.Second
)

type TokenConfig struct {
	Issuer    string
	Audience  string
	SecretKey []byte
}

type RefreshToken struct {
	bun.BaseModel `bun:"refresh_tokens"`

	ID         uuid.UUID       `bun:",pk,type:uuid"`
	GrantID    uuid.UUID       `bun:"grant_id,notnull"`
	Token      []byte          `bun:"-"`
	Key        uuid.UUID       `bun:"key,notnull"`
	TokenHash  password.Hashed `bun:"token_hash,notnull"`
	ExpiresAt  time.Time       `bun:"expires_at,notnull"`
	CreatedAt  time.Time       `bun:"created_at,notnull,nullzero"`
	ConsumedAt *time.Time      `bun:"consumed_at,nullzero"`
}

type RefreshTokenParts struct {
	Key   uuid.UUID
	Token []byte
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

	buf := make([]byte, refreshTokenDataLength)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	key, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token key: %w", err)
	}

	id, err := newTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token ID: %w", err)
	}

	hashed, err := password.Hash(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	return &RefreshToken{
		ID:        id,
		Token:     buf,
		Key:       key,
		TokenHash: hashed,
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

func SerializeRefreshToken(token *RefreshToken) ([]byte, error) {
	keyBytes, err := token.Key.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh token key: %w", err)
	}

	buf := make([]byte, 0, len(keyBytes)+len(token.Token))
	buf = append(buf, keyBytes...)
	buf = append(buf, token.Token...)

	return buf, nil
}

func DeserializeRefreshToken(buf []byte) (*RefreshTokenParts, error) {
	if len(buf) != refreshTokenLength {
		return nil, fmt.Errorf("refresh token is too short")
	}

	keyBytes := buf[:refreshTokenKeyLength]
	token := buf[refreshTokenKeyLength:]

	key, err := uuid.FromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token key: %w", err)
	}

	return &RefreshTokenParts{
		Key:   key,
		Token: token,
	}, nil
}
