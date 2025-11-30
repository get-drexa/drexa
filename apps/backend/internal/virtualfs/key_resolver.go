package virtualfs

import (
	"context"

	"github.com/get-drexa/drexa/internal/blob"
)

type BlobKeyResolver interface {
	KeyMode() blob.KeyMode
	Resolve(ctx context.Context, node *Node) (blob.Key, error)
	ResolveDeletionKeys(ctx context.Context, node *Node, allKeys []blob.Key) (*DeletionPlan, error)
}

type DeletionPlan struct {
	Prefix blob.Key
	Keys   []blob.Key
}
