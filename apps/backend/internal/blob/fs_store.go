package blob

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

var _ Store = &FSStore{}

type FSStore struct {
	config FSStoreConfig
}

type FSStoreConfig struct {
	Root string
}

func NewFSStore(config FSStoreConfig) *FSStore {
	return &FSStore{config: config}
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

func (s *FSStore) Retrieve(ctx context.Context, key Key) (io.ReadCloser, error) {
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

func (s *FSStore) Delete(ctx context.Context, key Key) error {
	err := os.Remove(filepath.Join(s.config.Root, string(key)))
	// no op if file does not exist
	// swallow error if file does not exist
	if err != nil && !os.IsNotExist(err) {
		return err
	}
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
