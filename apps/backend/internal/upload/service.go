package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
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

func (s *Service) CreateUpload(ctx context.Context, db bun.IDB, accountID uuid.UUID, opts CreateUploadOptions) (*Upload, error) {
	parentNode, err := s.vfs.FindNodeByPublicID(ctx, db, accountID, opts.ParentID)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if parentNode.Kind != virtualfs.NodeKindDirectory {
		return nil, ErrParentNotDirectory
	}

	node, err := s.vfs.CreateFile(ctx, db, accountID, virtualfs.CreateFileOptions{
		ParentID: parentNode.ID,
		Name:     opts.Name,
	})
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	var uploadURL string
	if s.blobStore.SupportsDirectUpload() {
		uploadURL, err = s.blobStore.GenerateUploadURL(ctx, node.BlobKey, blob.UploadURLOptions{
			Duration: 1 * time.Hour,
		})
		if err != nil {
			_ = s.vfs.PermanentlyDeleteNode(ctx, db, node)
			return nil, err
		}
	} else {
		uploadURL = ""
	}

	upload := &Upload{
		ID:         node.PublicID,
		Status:     StatusPending,
		TargetNode: node,
		UploadURL:  uploadURL,
	}

	s.pendingUploads.Store(upload.ID, upload)

	return upload, nil
}

func (s *Service) ReceiveUpload(ctx context.Context, db bun.IDB, accountID uuid.UUID, uploadID string, reader io.Reader) error {
	fmt.Printf("reader: %v\n", reader)
	n, ok := s.pendingUploads.Load(uploadID)
	if !ok {
		return ErrNotFound
	}

	upload, ok := n.(*Upload)
	if !ok {
		return ErrNotFound
	}

	if upload.TargetNode.AccountID != accountID {
		return ErrNotFound
	}

	err := s.vfs.WriteFile(ctx, db, upload.TargetNode, virtualfs.FileContentFromReader(reader))
	if err != nil {
		return err
	}

	upload.Status = StatusCompleted

	return nil
}

func (s *Service) CompleteUpload(ctx context.Context, db bun.IDB, accountID uuid.UUID, uploadID string) (*Upload, error) {
	n, ok := s.pendingUploads.Load(uploadID)
	if !ok {
		return nil, ErrNotFound
	}

	upload, ok := n.(*Upload)
	if !ok {
		return nil, ErrNotFound
	}

	if upload.TargetNode.AccountID != accountID {
		return nil, ErrNotFound
	}

	if upload.TargetNode.Status == virtualfs.NodeStatusReady && upload.Status == StatusCompleted {
		return upload, nil
	}

	err := s.vfs.WriteFile(ctx, db, upload.TargetNode, virtualfs.FileContentFromBlobKey(upload.TargetNode.BlobKey))
	if err != nil {
		if errors.Is(err, blob.ErrNotFound) {
			return nil, ErrContentNotUploaded
		}
		return nil, err
	}

	upload.Status = StatusCompleted
	s.pendingUploads.Delete(uploadID)

	return upload, nil
}
