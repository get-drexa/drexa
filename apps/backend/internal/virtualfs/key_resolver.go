package virtualfs

import (
	"context"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type BlobKeyResolver interface {
	KeyMode() blob.KeyMode
	Resolve(ctx context.Context, node *Node) (blob.Key, error)
}

type FlatKeyResolver struct{}

func NewFlatKeyResolver() *FlatKeyResolver {
	return &FlatKeyResolver{}
}

func (r *FlatKeyResolver) KeyMode() blob.KeyMode {
	return blob.KeyModeStable
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

type HierarchicalKeyResolver struct {
	db *bun.DB
}

func (r *HierarchicalKeyResolver) KeyMode() blob.KeyMode {
	return blob.KeyModeDerived
}

func (r *HierarchicalKeyResolver) Resolve(ctx context.Context, node *Node) (blob.Key, error) {
	path, err := buildNodeAbsolutePath(ctx, r.db, node.ID)
	if err != nil {
		return "", err
	}

	return blob.Key(path), nil
}
