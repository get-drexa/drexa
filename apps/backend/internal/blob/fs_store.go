package blob

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/get-drexa/drexa/internal/ioext"
)

var _ Store = &FSStore{}

type FSStore struct {
	config FSStoreConfig
}

type FSStoreConfig struct {
	Root      string
	UploadURL string
}

func NewFSStore(config FSStoreConfig) *FSStore {
	return &FSStore{config: config}
}

func (s *FSStore) Initialize(ctx context.Context) error {
	return os.MkdirAll(s.config.Root, 0755)
}

func (s *FSStore) GenerateUploadURL(ctx context.Context, key Key, opts UploadURLOptions) (string, error) {
	return s.config.UploadURL, nil
}

func (s *FSStore) Put(ctx context.Context, key Key, reader io.Reader) error {
	path := filepath.Join(s.config.Root, string(key))

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return ErrConflict
		}
		return err
	}

	defer f.Close()
	_, err = io.Copy(f, reader)
	if err != nil {
		_ = os.Remove(path)
		return err
	}

	return nil
}

func (s *FSStore) Read(ctx context.Context, key Key) (io.ReadCloser, error) {
	path := filepath.Join(s.config.Root, string(key))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *FSStore) ReadRange(ctx context.Context, key Key, offset, length int64) (io.ReadCloser, error) {
	path := filepath.Join(s.config.Root, string(key))

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return ioext.NewLimitReadCloser(f, length), nil
}

func (s *FSStore) ReadSize(ctx context.Context, key Key) (int64, error) {
	path := filepath.Join(s.config.Root, string(key))
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return fi.Size(), nil
}

func (s *FSStore) Delete(ctx context.Context, key Key) error {
	err := os.Remove(filepath.Join(s.config.Root, string(key)))
	// no op if file does not exist
	// swallow error if file does not exist
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *FSStore) Update(ctx context.Context, key Key, opts UpdateOptions) error {
	// Update is a no-op for FSStore
	return nil
}

func (s *FSStore) Move(ctx context.Context, srcKey, dstKey Key) error {
	oldPath := filepath.Join(s.config.Root, string(srcKey))
	newPath := filepath.Join(s.config.Root, string(dstKey))

	_, err := os.Stat(newPath)
	if err == nil {
		return ErrConflict
	}

	err = os.MkdirAll(filepath.Dir(newPath), 0755)
	if err != nil {
		return err
	}

	err = os.Rename(oldPath, newPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}

	return nil
}
