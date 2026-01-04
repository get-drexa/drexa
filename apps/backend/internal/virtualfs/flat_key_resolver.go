package virtualfs

import (
	"context"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type FlatKeyResolver struct{}

var _ BlobKeyResolver = &FlatKeyResolver{}

func NewFlatKeyResolver() *FlatKeyResolver {
	return &FlatKeyResolver{}
}

func (r *FlatKeyResolver) ShouldPersistKey() bool {
	return true
}

func (r *FlatKeyResolver) Resolve(ctx context.Context, db bun.IDB, node *Node) (blob.Key, error) {
	if node.BlobKey == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return "", err
		}
		return blob.Key(id.String()), nil
	}
	return node.BlobKey, nil
}

func (r *FlatKeyResolver) ResolveDeletionKeys(ctx context.Context, node *Node, allKeys []blob.Key) (*DeletionPlan, error) {
	return &DeletionPlan{Keys: allKeys}, nil
}

// ResolveBulkMoveOps returns nil for flat key storage since blob keys are UUIDs
// and don't change when nodes are moved to a different parent.
func (r *FlatKeyResolver) ResolveBulkMoveOps(ctx context.Context, db bun.IDB, nodes []*Node, newParentID uuid.UUID) ([]BlobMoveOp, error) {
	for _, node := range nodes[1:] {
		if node.ParentID != nodes[0].ParentID {
			return nil, ErrUnsupportedOperation
		}
	}
	return nil, nil
}

// ResolveRenameOp returns nil for flat key storage since blob keys are UUIDs
// and don't change when nodes are renamed.
func (r *FlatKeyResolver) ResolveRenameOp(ctx context.Context, db bun.IDB, node *Node, newName string) (*BlobMoveOp, error) {
	return nil, nil
}
