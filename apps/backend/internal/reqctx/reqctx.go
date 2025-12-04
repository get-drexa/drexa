package reqctx

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

const authenticatedUserKey = "authenticatedUser"

var ErrUnauthenticatedRequest = errors.New("unauthenticated request")

// AuthenticatedUser returns the authenticated user from the given fiber context.
// Returns ErrUnauthenticatedRequest if not authenticated.
// The caller must type assert the returned value to the appropriate user type.
func AuthenticatedUser(c *fiber.Ctx) any {
	return c.Locals(authenticatedUserKey)
}

// SetAuthenticatedUser sets the authenticated user in the fiber context.
func SetAuthenticatedUser(c *fiber.Ctx, user interface{}) {
	c.Locals(authenticatedUserKey, user)
}
