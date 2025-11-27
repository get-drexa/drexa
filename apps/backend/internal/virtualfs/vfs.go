package virtualfs

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"io"

	"github.com/gabriel-vasile/mimetype"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/google/uuid"
	"github.com/sqids/sqids-go"
	"github.com/uptrace/bun"
)

type VirtualFS struct {
	db          *bun.DB
	blobStore   blob.Store
	keyResolver BlobKeyResolver

	sqid *sqids.Sqids
}

type CreateNodeOptions struct {
	ParentID uuid.UUID
	Kind     NodeKind
	Name     string
}

type WriteFileOptions struct {
	ParentID uuid.UUID
	Name     string
}

func NewVirtualFS(db *bun.DB, blobStore blob.Store, keyResolver BlobKeyResolver) *VirtualFS {
	return &VirtualFS{
		db:          db,
		blobStore:   blobStore,
		keyResolver: keyResolver,
	}
}

func (vfs *VirtualFS) FindNode(ctx context.Context, userID, fileID string) (*Node, error) {
	var node Node
	err := vfs.db.NewSelect().Model(&node).
		Where("user_id = ?", userID).
		Where("id = ?", fileID).
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return &node, nil
}

func (vfs *VirtualFS) FindNodeByPublicID(ctx context.Context, userID uuid.UUID, publicID string) (*Node, error) {
	var node Node
	err := vfs.db.NewSelect().Model(&node).
		Where("user_id = ?", userID).
		Where("public_id = ?", publicID).
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return &node, nil
}

func (vfs *VirtualFS) WriteFile(ctx context.Context, userID uuid.UUID, reader io.Reader, opts WriteFileOptions) (*Node, error) {
	pid, err := vfs.generatePublicID()
	if err != nil {
		return nil, err
	}

	node := Node{
		PublicID: pid,
		UserID:   userID,
		ParentID: opts.ParentID,
		Kind:     NodeKindFile,
		Status:   NodeStatusPending,
		Name:     opts.Name,
	}

	_, err = vfs.db.NewInsert().Model(&node).Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrNodeConflict
		}
		return nil, err
	}

	cleanup := func() {
		_, _ = vfs.db.NewDelete().Model(&node).WherePK().Exec(ctx)
	}

	key, err := vfs.keyResolver.Resolve(ctx, &node)
	if err != nil {
		cleanup()
		return nil, err
	}

	h := make([]byte, 3072)
	n, err := io.ReadFull(reader, h)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		cleanup()
		return nil, err
	}
	h = h[:n]

	mt := mimetype.Detect(h)
	cr := NewCountingReader(io.MultiReader(bytes.NewReader(h), reader))

	err = vfs.blobStore.Put(ctx, key, cr)
	if err != nil {
		cleanup()
		return nil, err
	}

	node.BlobKey = key
	node.Size = cr.Count()
	node.MimeType = mt.String()
	node.Status = NodeStatusReady

	_, err = vfs.db.NewUpdate().Model(&node).
		Column("status", "blob_key", "size", "mime_type").
		WherePK().
		Exec(ctx)
	if err != nil {
		cleanup()
		return nil, err
	}

	return &node, nil
}

func (vfs *VirtualFS) CreateDirectory(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, name string) (*Node, error) {
	pid, err := vfs.generatePublicID()
	if err != nil {
		return nil, err
	}

	node := Node{
		PublicID: pid,
		UserID:   userID,
		ParentID: parentID,
		Kind:     NodeKindDirectory,
		Status:   NodeStatusReady,
		Name:     name,
	}

	_, err = vfs.db.NewInsert().Model(&node).Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrNodeConflict
		}
		return nil, err
	}

	return &node, nil
}

func (vfs *VirtualFS) SoftDeleteNode(ctx context.Context, node *Node) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	_, err := vfs.db.NewUpdate().Model(node).
		WherePK().
		Where("deleted_at IS NULL").
		Where("status = ?", NodeStatusReady).
		Set("deleted_at = NOW()").
		Returning("deleted_at").
		Exec(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNodeNotFound
		}
		return err
	}

	return nil
}

func (vfs *VirtualFS) RenameNode(ctx context.Context, node *Node, name string) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	_, err := vfs.db.NewUpdate().Model(node).
		WherePK().
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Set("name = ?", name).
		Returning("name, updated_at").
		Exec(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNodeNotFound
		}
		return err
	}
	return nil
}

func (vfs *VirtualFS) MoveNode(ctx context.Context, node *Node, parentID uuid.UUID) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	oldKey, err := vfs.keyResolver.Resolve(ctx, node)
	if err != nil {
		return err
	}

	_, err = vfs.db.NewUpdate().Model(node).
		WherePK().
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Set("parent_id = ?", parentID).
		Returning("parent_id, updated_at").
		Exec(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNodeNotFound
		}
		if database.IsUniqueViolation(err) {
			return ErrNodeConflict
		}
		return err
	}

	newKey, err := vfs.keyResolver.Resolve(ctx, node)
	if err != nil {
		return err
	}

	if node

	if oldKey != newKey {
		err = vfs.blobStore.Move(ctx, oldKey, newKey)
		if err != nil {
			return err
		}
		node.BlobKey = newKey
	}

	return nil
}

func (vfs *VirtualFS) generatePublicID() (string, error) {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint64(b[:])
	return vfs.sqid.Encode([]uint64{n})
}
