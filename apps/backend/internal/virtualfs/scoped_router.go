package virtualfs

import "github.com/gofiber/fiber/v2"

// ScopedRouter is a router that guarantees reqctx.VFSAccessScope(c)
// returns a valid *Scope for all registered routes.
//
// This is the base type for routers that provide VFS access scope.
// More specific router types may embed this
// to provide additional guarantees.
type ScopedRouter struct {
	fiber.Router
}
