package account

import "github.com/get-drexa/drexa/internal/virtualfs"

// ScopedRouter is a router with auth + account middleware applied.
// Routes registered on this router have access to:
//   - The authenticated user via reqctx.AuthenticatedUser()
//   - The current account via reqctx.CurrentAccount()
//   - The VFS scope via reqctx.VFSAccessScope()
//
// This embeds virtualfs.ScopedRouter, so it can be passed to functions
// that only require VFS scope by calling VFSRouter().
type ScopedRouter struct {
	virtualfs.ScopedRouter
}

// VFSRouter returns the embedded virtualfs.ScopedRouter for use with
// functions that only require VFS scope access.
func (r *ScopedRouter) VFSRouter() *virtualfs.ScopedRouter {
	return &r.ScopedRouter
}
