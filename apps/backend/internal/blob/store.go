package blob

import (
	"context"
	"io"
	"time"
)

type UploadURLOptions struct {
	Duration time.Duration
}

type UpdateOptions struct {
	ContentType string
}

type Store interface {
	GenerateUploadURL(ctx context.Context, key Key, opts UploadURLOptions) (string, error)
	Put(ctx context.Context, key Key, reader io.Reader) error
	Update(ctx context.Context, key Key, opts UpdateOptions) error
	Delete(ctx context.Context, key Key) error
	Move(ctx context.Context, srcKey, dstKey Key) error
	Read(ctx context.Context, key Key) (io.ReadCloser, error)
	ReadRange(ctx context.Context, key Key, offset, length int64) (io.ReadCloser, error)
	ReadSize(ctx context.Context, key Key) (int64, error)
}
