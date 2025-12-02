package blob

import (
	"context"
	"io"
	"time"
)

type UploadURLOptions struct {
	Duration time.Duration
}

type DownloadURLOptions struct {
	Duration time.Duration
}

type UpdateOptions struct {
	ContentType string
}

type Store interface {
	// SupportsDirectUpload returns true if the store allows files to be uploaded directly to the blob store.
	SupportsDirectUpload() bool

	// SupportsDirectDownload returns true if the store allows files to be downloaded directly from the blob store.
	SupportsDirectDownload() bool

	Initialize(ctx context.Context) error
	Put(ctx context.Context, key Key, reader io.Reader) error
	Update(ctx context.Context, key Key, opts UpdateOptions) error
	Delete(ctx context.Context, key Key) error
	DeletePrefix(ctx context.Context, prefix Key) error
	Move(ctx context.Context, srcKey, dstKey Key) error
	Read(ctx context.Context, key Key) (io.ReadCloser, error)
	ReadRange(ctx context.Context, key Key, offset, length int64) (io.ReadCloser, error)
	ReadSize(ctx context.Context, key Key) (int64, error)

	// GenerateUploadURL generates a URL that can be used to upload a file directly to the blob store. If unsupported, returns an empty string with no error.
	GenerateUploadURL(ctx context.Context, key Key, opts UploadURLOptions) (string, error)

	// GenerateDownloadURL generates a URL that can be used to download a file directly from the blob store. If unsupported, returns an empty string with no error.
	GenerateDownloadURL(ctx context.Context, key Key, opts DownloadURLOptions) (string, error)
}
