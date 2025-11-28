package upload

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
)

type Service struct {
	vfs       *virtualfs.VirtualFS
	blobStore blob.Store

	pendingUploads sync.Map
}

func NewService(vfs *virtualfs.VirtualFS, blobStore blob.Store) *Service {
	return &Service{
		vfs:       vfs,
		blobStore: blobStore,

		pendingUploads: sync.Map{},
	}
}

type CreateUploadOptions struct {
	ParentID string
	Name     string
}

type Upload struct {
	ID         string
	TargetNode *virtualfs.Node
	UploadURL  string
}

type CreateUploadResult struct {
	UploadID  string
	UploadURL string
}

func (s *Service) CreateUpload(ctx context.Context, userID uuid.UUID, opts CreateUploadOptions) (*Upload, error) {
	parentNode, err := s.vfs.FindNodeByPublicID(ctx, userID, opts.ParentID)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if parentNode.Kind != virtualfs.NodeKindDirectory {
		return nil, ErrParentNotDirectory
	}

	node, err := s.vfs.CreateFile(ctx, userID, virtualfs.CreateFileOptions{
		ParentID: parentNode.ID,
		Name:     opts.Name,
	})
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	uploadURL, err := s.blobStore.GenerateUploadURL(ctx, node.BlobKey, blob.UploadURLOptions{
		Duration: 1 * time.Hour,
	})
	if err != nil {
		return nil, err
	}

	upload := &Upload{
		ID:         node.PublicID,
		TargetNode: node,
		UploadURL:  uploadURL,
	}

	s.pendingUploads.Store(upload.ID, upload)

	return upload, nil
}

func (s *Service) ReceiveUpload(ctx context.Context, userID uuid.UUID, uploadID string, reader io.Reader) error {
	n, ok := s.pendingUploads.Load(uploadID)
	if !ok {
		return ErrNotFound
	}

	upload, ok := n.(*Upload)
	if !ok {
		return ErrNotFound
	}

	if upload.TargetNode.UserID != userID {
		return ErrNotFound
	}

	err := s.vfs.WriteFile(ctx, upload.TargetNode, virtualfs.FileContentFromReader(reader))
	if err != nil {
		return err
	}

	s.pendingUploads.Delete(uploadID)

	return nil
}

func (s *Service) CompleteUpload(ctx context.Context, userID uuid.UUID, uploadID string) error {
	n, ok := s.pendingUploads.Load(uploadID)
	if !ok {
		return ErrNotFound
	}

	upload, ok := n.(*Upload)
	if !ok {
		return ErrNotFound
	}

	if upload.TargetNode.UserID != userID {
		return ErrNotFound
	}

	if upload.TargetNode.Status == virtualfs.NodeStatusReady {
		return nil
	}

	err := s.vfs.WriteFile(ctx, upload.TargetNode, virtualfs.FileContentFromBlobKey(upload.TargetNode.BlobKey))
	if err != nil {
		return err
	}

	s.pendingUploads.Delete(uploadID)

	return nil
}
