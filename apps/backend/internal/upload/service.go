package upload

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/uptrace/bun"
)

type Service interface {
	CreateUpload(ctx context.Context, opts CreateUploadOptions, scope *virtualfs.Scope) (*Upload, error)
	ReceiveUpload(ctx context.Context, uploadID string, reader io.Reader, scope *virtualfs.Scope) error
	CompleteUpload(ctx context.Context, uploadID string, scope *virtualfs.Scope) (*Upload, error)
}

var _ Service = (*service)(nil)

type service struct {
	db        bun.IDB
	vfs       *virtualfs.VirtualFS
	blobStore blob.Store

	pendingUploads sync.Map
}

func NewService(db bun.IDB, vfs *virtualfs.VirtualFS, blobStore blob.Store) Service {
	return &service{
		db:        db,
		vfs:       vfs,
		blobStore: blobStore,

		pendingUploads: sync.Map{},
	}
}

type CreateUploadOptions struct {
	ParentID string
	Name     string
}

func (s *service) CreateUpload(ctx context.Context, opts CreateUploadOptions, scope *virtualfs.Scope) (*Upload, error) {
	if scope == nil {
		return nil, ErrUnauthorized
	}

	parentNode, err := s.vfs.FindNodeByPublicID(ctx, s.db, opts.ParentID, scope)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	if parentNode.Kind != virtualfs.NodeKindDirectory {
		return nil, ErrParentNotDirectory
	}

	node, err := s.vfs.CreateFile(ctx, s.db, virtualfs.CreateFileOptions{
		ParentID: parentNode.ID,
		Name:     opts.Name,
	}, scope)
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
			_ = s.vfs.PermanentlyDeleteNode(ctx, s.db, node, scope)

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

func (s *service) ReceiveUpload(ctx context.Context, uploadID string, reader io.Reader, scope *virtualfs.Scope) error {
	n, ok := s.pendingUploads.Load(uploadID)
	if !ok {
		return ErrNotFound
	}

	upload, ok := n.(*Upload)
	if !ok {
		return ErrNotFound
	}

	if scope == nil {
		return ErrUnauthorized
	}

	if upload.TargetNode.DriveID != scope.DriveID {
		return ErrNotFound
	}

	err := s.vfs.WriteFile(ctx, s.db, upload.TargetNode, virtualfs.FileContentFromReader(reader), scope)
	if err != nil {
		return err
	}

	upload.Status = StatusCompleted

	return nil
}

func (s *service) CompleteUpload(ctx context.Context, uploadID string, scope *virtualfs.Scope) (*Upload, error) {
	n, ok := s.pendingUploads.Load(uploadID)
	if !ok {
		return nil, ErrNotFound
	}

	upload, ok := n.(*Upload)
	if !ok {
		return nil, ErrNotFound
	}

	if scope == nil {
		return nil, ErrUnauthorized
	}

	if upload.TargetNode.DriveID != scope.DriveID {
		return nil, ErrNotFound
	}

	if upload.TargetNode.Status == virtualfs.NodeStatusReady && upload.Status == StatusCompleted {
		s.pendingUploads.Delete(uploadID)

		return upload, nil
	}

	err := s.vfs.WriteFile(ctx, s.db, upload.TargetNode, virtualfs.FileContentFromBlobKey(upload.TargetNode.BlobKey), scope)
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
