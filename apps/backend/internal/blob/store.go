package blob

import (
	"context"
	"io"
)

type Store interface {
	Put(ctx context.Context, key Key, reader io.Reader) error
	Retrieve(ctx context.Context, key Key) (io.ReadCloser, error)
	Delete(ctx context.Context, key Key) error
	Move(ctx context.Context, srcKey, dstKey Key) error
}
