package auth

import (
	"errors"
	"strings"

	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
)

const authenticatedUserKey = "authenticatedUser"

// NewBearerAuthMiddleware is a middleware that authenticates a request using a bearer token.
// To obtain the authenticated user in subsequent handlers, see AuthenticatedUser.
func NewBearerAuthMiddleware(s *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		token := parts[1]
		u, err := s.AuthenticateWithAccessToken(c.Context(), token)
		if err != nil {
			var e *InvalidAccessTokenError
			if errors.As(err, &e) {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			var nf *user.NotFoundError
			if errors.As(err, &nf) {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			return c.SendStatus(fiber.StatusInternalServerError)
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
