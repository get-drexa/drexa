package organization

import (
	"errors"
	"regexp"
	"strings"
)

const (
	slugMinLength = 1
	slugMaxLength = 63
)

// ReservedSlug is the special organization slug used to address the authenticated
// user's personal organization in API routes (e.g. /api/my/...).
const ReservedSlug = "my"

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

var ErrInvalidSlug = errors.New("invalid organization slug")

// NormalizeSlug lowercases and validates an organization slug.
func NormalizeSlug(input string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(input))
	if len(slug) < slugMinLength || len(slug) > slugMaxLength {
		return "", ErrInvalidSlug
	}

	if slug == ReservedSlug {
		return "", ErrInvalidSlug
	}

	if !slugPattern.MatchString(slug) {
		return "", ErrInvalidSlug
	}

	return slug, nil
}
