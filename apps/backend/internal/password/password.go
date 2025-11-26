package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Hashed represents a securely hashed password.
// This type ensures plaintext passwords cannot be accidentally stored.
type Hashed string

// argon2id parameters
const (
	memory      = 64 * 1024
	iterations  = 3
	parallelism = 2
	saltLength  = 16
	keyLength   = 32
)

var (
	ErrInvalidHash         = errors.New("invalid hash format")
	ErrIncompatibleHash    = errors.New("incompatible hash algorithm")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

type argon2Hash struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	salt        []byte
	hash        []byte
}

// Hash securely hashes a plaintext password using argon2id.
func Hash(plain string) (Hashed, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(plain),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		b64Salt,
		b64Hash,
	)

	return Hashed(encoded), nil
}

// Verify checks if a plaintext password matches a hashed password.
func Verify(plain string, hashed Hashed) (bool, error) {
	h, err := decodeHash(string(hashed))
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(plain),
		h.salt,
		h.iterations,
		h.memory,
		h.parallelism,
		uint32(len(h.hash)),
	)

	if subtle.ConstantTimeCompare(h.hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

func decodeHash(encodedHash string) (*argon2Hash, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, ErrIncompatibleHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}
	if version != argon2.Version {
		return nil, ErrIncompatibleVersion
	}

	h := &argon2Hash{}
	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&h.memory,
		&h.iterations,
		&h.parallelism,
	); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	h.salt = salt

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	h.hash = hash

	return h, nil
}
