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
