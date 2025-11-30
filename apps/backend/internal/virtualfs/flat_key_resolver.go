package virtualfs

import (
	"context"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/google/uuid"
)

type FlatKeyResolver struct{}

var _ BlobKeyResolver = &FlatKeyResolver{}

func NewFlatKeyResolver() *FlatKeyResolver {
	return &FlatKeyResolver{}
}

func (r *FlatKeyResolver) ShouldPersistKey() bool {
	return true
}

func (r *FlatKeyResolver) Resolve(ctx context.Context, node *Node) (blob.Key, error) {
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
