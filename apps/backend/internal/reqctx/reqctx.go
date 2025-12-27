package reqctx

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

const authenticatedUserKey = "authenticatedUser"
const vfsAccessScope = "vfsAccessScope"
const currentAccountKey = "currentAccount"

var ErrUnauthenticatedRequest = errors.New("unauthenticated request")

// AuthenticatedUser returns the authenticated user from the given fiber context.
// Returns ErrUnauthenticatedRequest if not authenticated.
// The caller must type assert the returned value to the appropriate user type.
func AuthenticatedUser(c *fiber.Ctx) any {
	return c.Locals(authenticatedUserKey)
}

// SetAuthenticatedUser sets the authenticated user in the fiber context.
func SetAuthenticatedUser(c *fiber.Ctx, user any) {
	c.Locals(authenticatedUserKey, user)
}

// SetCurrentAccount sets the current account in the fiber context.
func SetCurrentAccount(c *fiber.Ctx, account any) {
	c.Locals(currentAccountKey, account)
}

// SetVFSAccessScope sets the VFS access scope in the fiber context.
func SetVFSAccessScope(c *fiber.Ctx, scope any) {
	c.Locals(vfsAccessScope, scope)
}

// CurrentAccount returns the current account from the given fiber context.
func CurrentAccount(c *fiber.Ctx) any {
	return c.Locals(currentAccountKey)
}

// VFSAccessScope returns the VFS access scope from the given fiber context.
func VFSAccessScope(c *fiber.Ctx) any {
	return c.Locals(vfsAccessScope)
}
