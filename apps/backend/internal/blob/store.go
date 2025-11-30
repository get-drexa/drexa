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
	Initialize(ctx context.Context) error
	Put(ctx context.Context, key Key, reader io.Reader) error
	Update(ctx context.Context, key Key, opts UpdateOptions) error
	Delete(ctx context.Context, key Key) error
	DeletePrefix(ctx context.Context, prefix Key) error
	Move(ctx context.Context, srcKey, dstKey Key) error
	Read(ctx context.Context, key Key) (io.ReadCloser, error)
	ReadRange(ctx context.Context, key Key, offset, length int64) (io.ReadCloser, error)
	ReadSize(ctx context.Context, key Key) (int64, error)

	// SupportsDirectUpload returns true if the store allows files to be uploaded directly to the blob store.
	SupportsDirectUpload() bool

	// GenerateUploadURL generates a URL that can be used to upload a file directly to the blob store. If unsupported, returns an empty string with no error.
	GenerateUploadURL(ctx context.Context, key Key, opts UploadURLOptions) (string, error)
}
