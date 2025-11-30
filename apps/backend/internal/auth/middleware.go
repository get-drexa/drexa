package auth

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

const authenticatedUserKey = "authenticatedUser"

// NewBearerAuthMiddleware is a middleware that authenticates a request using a bearer token.
// To obtain the authenticated user in subsequent handlers, see AuthenticatedUser.
func NewBearerAuthMiddleware(s *Service, db *bun.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			slog.Info("no auth header")
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			slog.Info("invalid auth header")
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		token := parts[1]
		u, err := s.AuthenticateWithAccessToken(c.Context(), db, token)
		if err != nil {
			var e *InvalidAccessTokenError
			if errors.As(err, &e) {
				slog.Info("invalid access token")
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			var nf *user.NotFoundError
			if errors.As(err, &nf) {
				slog.Info("user not found")
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			return httperr.Internal(err)
		}

		c.Locals(authenticatedUserKey, u)

		return c.Next()
	}
}

// AuthenticatedUser returns the authenticated user from the given fiber context.
// Returns ErrUnauthenticatedRequest if not authenticated.
func AuthenticatedUser(c *fiber.Ctx) (*user.User, error) {
	if u, ok := c.Locals(authenticatedUserKey).(*user.User); ok {
		return u, nil
	}
	return nil, ErrUnauthenticatedRequest
}
