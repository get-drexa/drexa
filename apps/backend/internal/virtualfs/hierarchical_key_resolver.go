package virtualfs

import (
	"context"
	"fmt"

	"github.com/get-drexa/drexa/internal/blob"
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

func (r *HierarchicalKeyResolver) Resolve(ctx context.Context, node *Node) (blob.Key, error) {
	path, err := buildNodeAbsolutePath(ctx, r.db, node.ID)
	if err != nil {
		return "", err
	}

	return blob.Key(fmt.Sprintf("%s/%s", node.AccountID, path)), nil
}

func (r *HierarchicalKeyResolver) ResolveDeletionKeys(ctx context.Context, node *Node, allKeys []blob.Key) (*DeletionPlan, error) {
	path, err := buildNodeAbsolutePath(ctx, r.db, node.ID)
	if err != nil {
		return nil, err
	}
	return &DeletionPlan{Prefix: blob.Key(path)}, nil
}
