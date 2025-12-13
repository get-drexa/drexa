package virtualfs

import (
	"context"
	"fmt"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HierarchicalKeyResolver struct {
	db *bun.DB
}

var _ BlobKeyResolver = &HierarchicalKeyResolver{}

func NewHierarchicalKeyResolver(db *bun.DB) *HierarchicalKeyResolver {
	return &HierarchicalKeyResolver{db: db}
}

func (r *HierarchicalKeyResolver) ShouldPersistKey() bool {
	return false
}

func (r *HierarchicalKeyResolver) Resolve(ctx context.Context, db bun.IDB, node *Node) (blob.Key, error) {
	path, err := buildNodeAbsolutePathString(ctx, db, node)
	if err != nil {
		return "", err
	}

	return blob.Key(fmt.Sprintf("%s/%s", node.AccountID, path)), nil
}

func (r *HierarchicalKeyResolver) ResolveDeletionKeys(ctx context.Context, node *Node, allKeys []blob.Key) (*DeletionPlan, error) {
	path, err := buildNodeAbsolutePathString(ctx, r.db, node)
	if err != nil {
		return nil, err
	}
	return &DeletionPlan{Prefix: blob.Key(path)}, nil
}

// ResolveBulkMoveOps computes blob move operations for nodes being moved to a new parent.
// This implementation optimizes by computing parent paths only once (2 queries total),
// rather than computing the full path for each node individually (N queries).
func (r *HierarchicalKeyResolver) ResolveBulkMoveOps(ctx context.Context, db bun.IDB, nodes []*Node, newParentID uuid.UUID) ([]BlobMoveOp, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	accountID := nodes[0].AccountID
	oldParentID := nodes[0].ParentID

	for _, node := range nodes[1:] {
		if node.ParentID != oldParentID {
			return nil, ErrUnsupportedOperation
		}
	}

	oldParentPath, err := buildPathFromNodeID(ctx, db, oldParentID)
	if err != nil {
		return nil, err
	}

	newParentPath, err := buildPathFromNodeID(ctx, db, newParentID)
	if err != nil {
		return nil, err
	}

	// For each node, construct old and new keys using the precomputed parent paths
	ops := make([]BlobMoveOp, len(nodes))
	for i, node := range nodes {
		oldKey := blob.Key(fmt.Sprintf("%s/%s/%s", accountID, oldParentPath, node.Name))
		newKey := blob.Key(fmt.Sprintf("%s/%s/%s", accountID, newParentPath, node.Name))
		ops[i] = BlobMoveOp{OldKey: oldKey, NewKey: newKey}
	}

	return ops, nil
}
