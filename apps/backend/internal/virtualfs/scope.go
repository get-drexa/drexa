package virtualfs

import "github.com/google/uuid"

// Scope defines the bounded view of the virtual filesystem that a caller is allowed to operate on.
// It is populated by higher layers (account/share middleware) and enforced by VFS methods.
type Scope struct {
	// DriveID is the owner of the storage (the tenant). It stays constant even when a share actor accesses it.
	DriveID uuid.UUID

	// RootNodeID is the top-most node the caller is allowed to traverse; all accesses must stay under it.
	// It must be set for all VFS access operations.
	RootNodeID uuid.UUID

	// AllowedOps lists which operations this scope may perform (read, write, delete, etc).
	AllowedOps map[Operation]bool

	// AllowedNodes is an optional allowlist of node IDs permitted within RootNodeID.
	// When nil or empty, the full subtree is allowed; when set, only allowlisted nodes (and descendants) are allowed.
	AllowedNodes map[uuid.UUID]struct{}

	// ActorKind identifies who performs the action (user vs share link) for auditing.
	ActorKind ScopeActorKind

	// ActorID is the identifier of the actor (user ID, share ID, etc).
	ActorID uuid.UUID
}

var AllAllowedOps = map[Operation]bool{
	OperationRead:   true,
	OperationWrite:  true,
	OperationDelete: true,
	OperationUpload: true,
	OperationShare:  true,
}

// Allows reports whether the scope permits the given operation.
func (s *Scope) Allows(op Operation) bool {
	return s != nil && s.AllowedOps[op]
}

// Operation enumerates supported actions.
type Operation string

const (
	OperationRead   Operation = "read"
	OperationWrite  Operation = "write"
	OperationDelete Operation = "delete"
	OperationUpload Operation = "upload"
	OperationShare  Operation = "share"
)

// ScopeActorKind labels the type of actor behind the request.
type ScopeActorKind string

const (
	ScopeActorAccount ScopeActorKind = "account"
	ScopeActorShare   ScopeActorKind = "share"
)
