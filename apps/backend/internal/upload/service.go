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

func (s *Service) CreateUpload(ctx context.Context, db bun.IDB, opts CreateUploadOptions, scope *virtualfs.Scope) (*Upload, error) {
	if scope == nil {
		return nil, ErrUnauthorized
	}

	parentNode, err := s.vfs.FindNodeByPublicID(ctx, db, opts.ParentID, scope)
	if err != nil {
		if errors.Is(err, virtualfs.ErrNodeNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if parentNode.Kind != virtualfs.NodeKindDirectory {
		return nil, ErrParentNotDirectory
	}

	node, err := s.vfs.CreateFile(ctx, db, virtualfs.CreateFileOptions{
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
			_ = s.vfs.PermanentlyDeleteNode(ctx, db, node, scope)
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

func (s *Service) ReceiveUpload(ctx context.Context, db bun.IDB, uploadID string, reader io.Reader, scope *virtualfs.Scope) error {
	fmt.Printf("reader: %v\n", reader)
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

	err := s.vfs.WriteFile(ctx, db, upload.TargetNode, virtualfs.FileContentFromReader(reader), scope)
	if err != nil {
		return err
	}

	upload.Status = StatusCompleted

	return nil
}

func (s *Service) CompleteUpload(ctx context.Context, db bun.IDB, uploadID string, scope *virtualfs.Scope) (*Upload, error) {
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

	err := s.vfs.WriteFile(ctx, db, upload.TargetNode, virtualfs.FileContentFromBlobKey(upload.TargetNode.BlobKey), scope)
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
