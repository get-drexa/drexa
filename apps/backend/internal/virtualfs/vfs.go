package virtualfs

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/ioext"
	"github.com/google/uuid"
	"github.com/sqids/sqids-go"
	"github.com/uptrace/bun"
)

type VirtualFS struct {
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

const RootDirectoryName = "root"

func New(blobStore blob.Store, keyResolver BlobKeyResolver) (*VirtualFS, error) {
	sqid, err := sqids.New()
	if err != nil {
		return nil, err
	}
	return &VirtualFS{
		blobStore:   blobStore,
		keyResolver: keyResolver,
		sqid:        sqid,
	}, nil
}

func (vfs *VirtualFS) FindNode(ctx context.Context, db bun.IDB, accountID, fileID string) (*Node, error) {
	var node Node
	err := db.NewSelect().Model(&node).
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

func (vfs *VirtualFS) FindNodeByPublicID(ctx context.Context, db bun.IDB, accountID uuid.UUID, publicID string) (*Node, error) {
	var node Node
	err := db.NewSelect().Model(&node).
		Where("account_id = ?", accountID).
		Where("public_id = ?", publicID).
		Where("status = ?", NodeStatusReady).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return &node, nil
}

func (vfs *VirtualFS) ListChildren(ctx context.Context, db bun.IDB, node *Node) ([]*Node, error) {
	if !node.IsAccessible() {
		return nil, ErrNodeNotFound
	}

	var nodes []*Node
	err := db.NewSelect().Model(&nodes).
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

func (vfs *VirtualFS) CreateFile(ctx context.Context, db bun.IDB, accountID uuid.UUID, opts CreateFileOptions) (*Node, error) {
	pid, err := vfs.generatePublicID()
	if err != nil {
		return nil, err
	}

	id, err := newNodeID()
	if err != nil {
		return nil, err
	}

	node := Node{
		ID:        id,
		PublicID:  pid,
		AccountID: accountID,
		ParentID:  opts.ParentID,
		Kind:      NodeKindFile,
		Status:    NodeStatusPending,
		Name:      opts.Name,
	}

	if vfs.keyResolver.ShouldPersistKey() {
		node.BlobKey, err = vfs.keyResolver.Resolve(ctx, db, &node)
		if err != nil {
			return nil, err
		}
	}

	_, err = db.NewInsert().Model(&node).Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrNodeConflict
		}
		return nil, err
	}

	return &node, nil
}

func (vfs *VirtualFS) WriteFile(ctx context.Context, db bun.IDB, node *Node, content FileContent) error {
	if content.Reader == nil && content.BlobKey.IsNil() {
		return blob.ErrInvalidFileContent
	}

	if node.DeletedAt != nil {
		return ErrNodeNotFound
	}

	setCols := make([]string, 0, 4)

	if content.Reader != nil {
		key, err := vfs.keyResolver.Resolve(ctx, db, node)
		if err != nil {
			return err
		}

		buf := make([]byte, 3072)
		n, err := io.ReadFull(content.Reader, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}
		buf = buf[:n]

		mt := mimetype.Detect(buf)
		cr := ioext.NewCountingReader(io.MultiReader(bytes.NewReader(buf), content.Reader))

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
		node.BlobKey = content.BlobKey

		b, err := vfs.blobStore.ReadRange(ctx, content.BlobKey, 0, 3072)
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

		s, err := vfs.blobStore.ReadSize(ctx, content.BlobKey)
		if err != nil {
			return err
		}
		node.Size = s

		setCols = append(setCols, "mime_type", "blob_key", "size", "status")
	}

	_, err := db.NewUpdate().Model(node).
		Column(setCols...).
		WherePK().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (vfs *VirtualFS) ReadFile(ctx context.Context, db bun.IDB, node *Node) (FileContent, error) {
	if node.Kind != NodeKindFile {
		return EmptyFileContent(), ErrUnsupportedOperation
	}

	key, err := vfs.keyResolver.Resolve(ctx, db, node)
	if err != nil {
		return EmptyFileContent(), err
	}

	if vfs.blobStore.SupportsDirectDownload() {
		url, err := vfs.blobStore.GenerateDownloadURL(ctx, key, blob.DownloadURLOptions{
			Duration: 1 * time.Hour,
		})
		if err != nil {
			return EmptyFileContent(), err
		}
		return FileContentFromURL(url), nil
	}

	reader, err := vfs.blobStore.Read(ctx, key)
	if err != nil {
		return EmptyFileContent(), err
	}

	return FileContentFromReaderWithSize(reader, node.Size), nil
}

func (vfs *VirtualFS) CreateDirectory(ctx context.Context, db bun.IDB, accountID uuid.UUID, parentID uuid.UUID, name string) (*Node, error) {
	pid, err := vfs.generatePublicID()
	if err != nil {
		return nil, err
	}

	id, err := newNodeID()
	if err != nil {
		return nil, err
	}

	node := &Node{
		ID:        id,
		PublicID:  pid,
		AccountID: accountID,
		ParentID:  parentID,
		Kind:      NodeKindDirectory,
		Status:    NodeStatusReady,
		Name:      name,
	}

	_, err = db.NewInsert().Model(node).Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrNodeConflict
		}
		return nil, err
	}

	return node, nil
}

func (vfs *VirtualFS) SoftDeleteNode(ctx context.Context, db bun.IDB, node *Node) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	_, err := db.NewUpdate().Model(node).
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

func (vfs *VirtualFS) RestoreNode(ctx context.Context, db bun.IDB, node *Node) error {
	if node.Status != NodeStatusReady {
		return ErrNodeNotFound
	}

	_, err := db.NewUpdate().Model(node).
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

func (vfs *VirtualFS) RenameNode(ctx context.Context, db bun.IDB, node *Node, name string) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	oldKey, err := vfs.keyResolver.Resolve(ctx, db, node)
	if err != nil {
		return err
	}

	_, err = db.NewUpdate().Model(node).
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

	newKey, err := vfs.keyResolver.Resolve(ctx, db, node)
	if err != nil {
		return err
	}

	if oldKey != newKey {
		err = vfs.blobStore.Move(ctx, oldKey, newKey)
		if err != nil {
			return err
		}

		if vfs.keyResolver.ShouldPersistKey() {
			node.BlobKey = newKey
			_, err = db.NewUpdate().Model(node).
				WherePK().
				Set("blob_key = ?", newKey).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vfs *VirtualFS) MoveNode(ctx context.Context, db bun.IDB, node *Node, parentID uuid.UUID) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	oldKey, err := vfs.keyResolver.Resolve(ctx, db, node)
	if err != nil {
		return err
	}

	_, err = db.NewUpdate().Model(node).
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

	newKey, err := vfs.keyResolver.Resolve(ctx, db, node)
	if err != nil {
		return err
	}

	err = vfs.blobStore.Move(ctx, oldKey, newKey)
	if err != nil {
		return err
	}

	if vfs.keyResolver.ShouldPersistKey() {
		node.BlobKey = newKey
		_, err = db.NewUpdate().Model(node).
			WherePK().
			Set("blob_key = ?", newKey).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vfs *VirtualFS) AbsolutePath(ctx context.Context, db bun.IDB, node *Node) (string, error) {
	if !node.IsAccessible() {
		return "", ErrNodeNotFound
	}
	return buildNodeAbsolutePathString(ctx, db, node)
}

func (vfs *VirtualFS) PermanentlyDeleteNode(ctx context.Context, db bun.IDB, node *Node) error {
	switch node.Kind {
	case NodeKindFile:
		return vfs.permanentlyDeleteFileNode(ctx, db, node)
	case NodeKindDirectory:
		return vfs.permanentlyDeleteDirectoryNode(ctx, db, node)
	default:
		return ErrUnsupportedOperation
	}
}

func (vfs *VirtualFS) permanentlyDeleteFileNode(ctx context.Context, db bun.IDB, node *Node) error {
	key, err := vfs.keyResolver.Resolve(ctx, db, node)
	if err != nil {
		return err
	}

	err = vfs.blobStore.Delete(ctx, key)
	if err != nil {
		return err
	}

	_, err = db.NewDelete().Model(node).WherePK().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (vfs *VirtualFS) permanentlyDeleteDirectoryNode(ctx context.Context, db bun.IDB, node *Node) error {
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

	// If db is already a transaction, use it directly; otherwise start a new transaction
	var tx bun.IDB
	var startedTx *bun.Tx
	switch v := db.(type) {
	case *bun.DB:
		newTx, err := v.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		startedTx = &newTx
		tx = newTx
		defer func() {
			if startedTx != nil {
				(*startedTx).Rollback()
			}
		}()
	default:
		// Assume it's already a transaction
		tx = db
	}

	var records []nodeRecord
	err := tx.NewRaw(descendantsQuery, node.ID).Scan(ctx, &records)
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

	// Only commit if we started the transaction
	if startedTx != nil {
		err := (*startedTx).Commit()
		startedTx = nil // Prevent defer from rolling back
		return err
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

func (vfs *VirtualFS) RealPath(ctx context.Context, db bun.IDB, node *Node) (Path, error) {
	if !node.IsAccessible() {
		return nil, ErrNodeNotFound
	}
	return buildNoteAbsolutePath(ctx, db, node)
}
