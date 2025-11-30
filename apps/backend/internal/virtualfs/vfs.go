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
	"github.com/get-drexa/drexa/internal/ioext"
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

type CreateFileOptions struct {
	ParentID uuid.UUID
	Name     string
}

type FileContent struct {
	reader  io.Reader
	blobKey blob.Key
}

func FileContentFromReader(reader io.Reader) FileContent {
	return FileContent{reader: reader}
}

func FileContentFromBlobKey(blobKey blob.Key) FileContent {
	return FileContent{blobKey: blobKey}
}

func NewVirtualFS(db *bun.DB, blobStore blob.Store, keyResolver BlobKeyResolver) (*VirtualFS, error) {
	sqid, err := sqids.New()
	if err != nil {
		return nil, err
	}
	return &VirtualFS{
		db:          db,
		blobStore:   blobStore,
		keyResolver: keyResolver,
		sqid:        sqid,
	}, nil
}

func (vfs *VirtualFS) FindNode(ctx context.Context, accountID, fileID string) (*Node, error) {
	var node Node
	err := vfs.db.NewSelect().Model(&node).
		Where("account_id = ?", accountID).
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

func (vfs *VirtualFS) FindNodeByPublicID(ctx context.Context, accountID uuid.UUID, publicID string) (*Node, error) {
	var node Node
	err := vfs.db.NewSelect().Model(&node).
		Where("account_id = ?", accountID).
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

func (vfs *VirtualFS) ListChildren(ctx context.Context, node *Node) ([]*Node, error) {
	if !node.IsAccessible() {
		return nil, ErrNodeNotFound
	}

	var nodes []*Node
	err := vfs.db.NewSelect().Model(&nodes).
		Where("account_id = ?", node.AccountID).
		Where("parent_id = ?", node.ID).
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Node, 0), nil
		}
		return nil, err
	}

	return nodes, nil
}

func (vfs *VirtualFS) CreateFile(ctx context.Context, accountID uuid.UUID, opts CreateFileOptions) (*Node, error) {
	pid, err := vfs.generatePublicID()
	if err != nil {
		return nil, err
	}

	node := Node{
		PublicID:  pid,
		AccountID: accountID,
		ParentID:  opts.ParentID,
		Kind:      NodeKindFile,
		Status:    NodeStatusPending,
		Name:      opts.Name,
	}

	if vfs.keyResolver.ShouldPersistKey() {
		node.BlobKey, err = vfs.keyResolver.Resolve(ctx, &node)
		if err != nil {
			return nil, err
		}
	}

	_, err = vfs.db.NewInsert().Model(&node).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrNodeConflict
		}
		return nil, err
	}

	return &node, nil
}

func (vfs *VirtualFS) WriteFile(ctx context.Context, node *Node, content FileContent) error {
	if content.reader == nil && content.blobKey.IsNil() {
		return blob.ErrInvalidFileContent
	}

	if !node.DeletedAt.IsZero() {
		return ErrNodeNotFound
	}

	setCols := make([]string, 0, 4)

	if content.reader != nil {
		key, err := vfs.keyResolver.Resolve(ctx, node)
		if err != nil {
			return err
		}

		buf := make([]byte, 3072)
		n, err := io.ReadFull(content.reader, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}
		buf = buf[:n]

		mt := mimetype.Detect(buf)
		cr := ioext.NewCountingReader(io.MultiReader(bytes.NewReader(buf), content.reader))

		err = vfs.blobStore.Put(ctx, key, cr)
		if err != nil {
			return err
		}

		if vfs.keyResolver.ShouldPersistKey() {
			node.BlobKey = key
			setCols = append(setCols, "blob_key")
		}

		node.MimeType = mt.String()
		node.Size = cr.Count()
		node.Status = NodeStatusReady

		setCols = append(setCols, "mime_type", "size", "status")
	} else {
		node.BlobKey = content.blobKey

		b, err := vfs.blobStore.ReadRange(ctx, content.blobKey, 0, 3072)
		if err != nil {
			return err
		}
		defer b.Close()

		buf := make([]byte, 3072)
		n, err := io.ReadFull(b, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}
		buf = buf[:n]
		mt := mimetype.Detect(buf)
		node.MimeType = mt.String()
		node.Status = NodeStatusReady

		s, err := vfs.blobStore.ReadSize(ctx, content.blobKey)
		if err != nil {
			return err
		}
		node.Size = s

		setCols = append(setCols, "mime_type", "blob_key", "size", "status")
	}

	_, err := vfs.db.NewUpdate().Model(&node).
		Column(setCols...).
		WherePK().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (vfs *VirtualFS) CreateDirectory(ctx context.Context, accountID uuid.UUID, parentID uuid.UUID, name string) (*Node, error) {
	pid, err := vfs.generatePublicID()
	if err != nil {
		return nil, err
	}

	node := Node{
		PublicID:  pid,
		AccountID: accountID,
		ParentID:  parentID,
		Kind:      NodeKindDirectory,
		Status:    NodeStatusReady,
		Name:      name,
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

func (vfs *VirtualFS) RestoreNode(ctx context.Context, node *Node) error {
	if node.Status != NodeStatusReady {
		return ErrNodeNotFound
	}

	_, err := vfs.db.NewUpdate().Model(node).
		WherePK().
		Where("deleted_at IS NOT NULL").
		Set("deleted_at = NULL").
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

	err = vfs.blobStore.Move(ctx, oldKey, newKey)
	if err != nil {
		return err
	}

	if vfs.keyResolver.ShouldPersistKey() {
		node.BlobKey = newKey
		_, err = vfs.db.NewUpdate().Model(node).
			WherePK().
			Set("blob_key = ?", newKey).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vfs *VirtualFS) AbsolutePath(ctx context.Context, node *Node) (string, error) {
	if !node.IsAccessible() {
		return "", ErrNodeNotFound
	}
	return buildNodeAbsolutePath(ctx, vfs.db, node.ID)
}

func (vfs *VirtualFS) PermanentlyDeleteNode(ctx context.Context, node *Node) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}
	switch node.Kind {
	case NodeKindFile:
		return vfs.permanentlyDeleteFileNode(ctx, node)
	case NodeKindDirectory:
		return vfs.permanentlyDeleteDirectoryNode(ctx, node)
	default:
		return ErrUnsupportedOperation
	}
}

func (vfs *VirtualFS) permanentlyDeleteFileNode(ctx context.Context, node *Node) error {
	err := vfs.blobStore.Delete(ctx, node.BlobKey)
	if err != nil {
		return err
	}

	_, err = vfs.db.NewDelete().Model(node).WherePK().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (vfs *VirtualFS) permanentlyDeleteDirectoryNode(ctx context.Context, node *Node) error {
	const descendantsQuery = `WITH RECURSIVE descendants AS (
		SELECT id, blob_key FROM vfs_nodes WHERE id = ?
		UNION ALL
		SELECT n.id, n.blob_key FROM vfs_nodes n
		JOIN descendants d ON n.parent_id = d.id
	)
	SELECT id, blob_key FROM descendants`

	type nodeRecord struct {
		ID      uuid.UUID `bun:"id"`
		BlobKey blob.Key  `bun:"blob_key"`
	}

	tx, err := vfs.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var records []nodeRecord
	err = tx.NewRaw(descendantsQuery, node.ID).Scan(ctx, &records)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNodeNotFound
		}
		return err
	}

	if len(records) == 0 {
		return ErrNodeNotFound
	}

	nodeIDs := make([]uuid.UUID, 0, len(records))
	blobKeys := make([]blob.Key, 0, len(records))
	for _, r := range records {
		nodeIDs = append(nodeIDs, r.ID)
		if !r.BlobKey.IsNil() {
			blobKeys = append(blobKeys, r.BlobKey)
		}
	}

	plan, err := vfs.keyResolver.ResolveDeletionKeys(ctx, node, blobKeys)
	if err != nil {
		return err
	}

	_, err = tx.NewDelete().
		Model((*Node)(nil)).
		Where("id IN (?)", bun.In(nodeIDs)).
		Exec(ctx)
	if err != nil {
		return err
	}

	if !plan.Prefix.IsNil() {
		_ = vfs.blobStore.DeletePrefix(ctx, plan.Prefix)
	} else {
		for _, key := range plan.Keys {
			_ = vfs.blobStore.Delete(ctx, key)
		}
	}

	return tx.Commit()
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
