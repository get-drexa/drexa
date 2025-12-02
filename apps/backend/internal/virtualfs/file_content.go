package virtualfs

import (
	"io"

	"github.com/get-drexa/drexa/internal/blob"
)

type FileContent struct {
	Size    int64
	Reader  io.ReadCloser
	BlobKey blob.Key
	URL     string
}

func EmptyFileContent() FileContent {
	return FileContent{}
}

func FileContentFromReader(reader io.Reader) FileContent {
	return FileContent{Reader: io.NopCloser(reader)}
}

func FileContentFromReaderWithSize(reader io.Reader, size int64) FileContent {
	return FileContent{Reader: io.NopCloser(reader), Size: size}
}

func FileContentFromBlobKey(blobKey blob.Key) FileContent {
	return FileContent{BlobKey: blobKey}
}

func FileContentFromURL(url string) FileContent {
	return FileContent{URL: url}
}
