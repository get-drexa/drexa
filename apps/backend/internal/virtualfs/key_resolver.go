package virtualfs

import (
	"context"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type BlobKeyResolver interface {
	// ShouldPersistKey returns true if the resolved key should be stored in node.BlobKey.
	// Flat keys (e.g. UUIDs) return true - key is generated once and stored.
	// Hierarchical keys return false - key is derived from path each time.
	ShouldPersistKey() bool
	Resolve(ctx context.Context, db bun.IDB, node *Node) (blob.Key, error)
	ResolveDeletionKeys(ctx context.Context, node *Node, allKeys []blob.Key) (*DeletionPlan, error)

	// ResolveBulkMoveOps returns blob move operations for nodes being moved to a new parent.
	// Returns ErrBulkMoveRequiresSameParent if nodes don't all share the same parent.
	// Returns nil, nil if no blob moves are needed (e.g., flat key storage where keys are UUIDs).
	ResolveBulkMoveOps(ctx context.Context, db bun.IDB, nodes []*Node, newParentID uuid.UUID) ([]BlobMoveOp, error)

	// ResolveRenameOp returns the blob move operation needed when renaming a node.
	// Returns nil, nil if no blob move is needed (e.g., flat key storage where keys are UUIDs).
	// For hierarchical storage, returns a single move op for both files and directories
	// (os.Rename handles directories atomically).
	ResolveRenameOp(ctx context.Context, db bun.IDB, node *Node, newName string) (*BlobMoveOp, error)
}

type DeletionPlan struct {
	Prefix blob.Key
	Keys   []blob.Key
}

// BlobMoveOp represents a blob move operation from OldKey to NewKey.
type BlobMoveOp struct {
	Node   *Node
	OldKey blob.Key
	NewKey blob.Key
}
