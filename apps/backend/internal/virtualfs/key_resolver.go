package virtualfs

import (
	"context"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/uptrace/bun"
)

type BlobKeyResolver interface {
	// ShouldPersistKey returns true if the resolved key should be stored in node.BlobKey.
	// Flat keys (e.g. UUIDs) return true - key is generated once and stored.
	// Hierarchical keys return false - key is derived from path each time.
	ShouldPersistKey() bool
	Resolve(ctx context.Context, db bun.IDB, node *Node) (blob.Key, error)
	ResolveDeletionKeys(ctx context.Context, node *Node, allKeys []blob.Key) (*DeletionPlan, error)
}

type DeletionPlan struct {
	Prefix blob.Key
	Keys   []blob.Key
}
