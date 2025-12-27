package virtualfs

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
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

type ListChildrenOrder string

const (
	ListChildrenOrderByName      ListChildrenOrder = "name"
	ListChildrenOrderByCreatedAt ListChildrenOrder = "created_at"
	ListChildrenOrderByUpdatedAt ListChildrenOrder = "updated_at"
)

type ListChildrenDirection int

const (
	ListChildrenDirectionAsc ListChildrenDirection = iota
	ListChildrenDirectionDesc
)

const listChildrenDefaultLimit = 50

type VirtualFS struct {
	blobStore   blob.Store
	keyResolver BlobKeyResolver

	sqid *sqids.Sqids
}

type CreateFileOptions struct {
	ParentID uuid.UUID
	Name     string
}

type MoveFileError struct {
	Node  *Node
	Error error
}

type MoveFilesResult struct {
	Moved     []*Node
	Conflicts []*Node
	Errors    []MoveFileError
}

type ListChildrenOptions struct {
	Limit          int
	OrderBy        ListChildrenOrder
	OrderDirection ListChildrenDirection
	Cursor         *ListChildrenCursor
}

type ListChildrenCursor struct {
	Node           *Node
	OrderBy        ListChildrenOrder
	OrderDirection ListChildrenDirection
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

func (vfs *VirtualFS) FindNode(ctx context.Context, db bun.IDB, fileID string, scope *Scope) (*Node, error) {
	if !isScopeSet(scope) {
		return nil, ErrAccessDenied
	}

	var node Node
	err := db.NewSelect().Model(&node).
		Where("account_id = ?", scope.AccountID).
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

	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationRead, node.ID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrAccessDenied
	}
	return &node, nil
}

func (vfs *VirtualFS) FindNodeByPublicID(ctx context.Context, db bun.IDB, publicID string, scope *Scope) (*Node, error) {
	nodes, err := vfs.FindNodesByPublicID(ctx, db, []string{publicID}, scope)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, ErrNodeNotFound
	}
	return nodes[0], nil
}

func (vfs *VirtualFS) FindNodesByPublicID(ctx context.Context, db bun.IDB, publicIDs []string, scope *Scope) ([]*Node, error) {
	if len(publicIDs) == 0 {
		return nil, nil
	}
	if !isScopeSet(scope) {
		return nil, ErrAccessDenied
	}

	var nodes []*Node
	err := db.NewSelect().Model(&nodes).
		Where("account_id = ?", scope.AccountID).
		Where("public_id IN (?)", bun.In(publicIDs)).
		Where("status = ?", NodeStatusReady).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return vfs.filterNodesByScope(ctx, db, scope, nodes)
}

func (vfs *VirtualFS) FindRootDirectory(ctx context.Context, db bun.IDB, accountID uuid.UUID) (*Node, error) {
	root := new(Node)

	err := db.NewSelect().Model(root).
		Where("account_id = ?", accountID).
		Where("parent_id IS NULL").
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	if root.Kind != NodeKindDirectory {
		return nil, ErrNodeNotFound
	}

	return root, nil
}

// CreateRootDirectory creates the account root directory node.
func (vfs *VirtualFS) CreateRootDirectory(ctx context.Context, db bun.IDB, accountID uuid.UUID) (*Node, error) {
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
		ParentID:  uuid.Nil,
		Kind:      NodeKindDirectory,
		Status:    NodeStatusReady,
		Name:      RootDirectoryName,
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

// ListChildren returns the children of a directory node with optional sorting and cursor-based pagination.
func (vfs *VirtualFS) ListChildren(ctx context.Context, db bun.IDB, node *Node, opts ListChildrenOptions, scope *Scope) ([]*Node, *ListChildrenCursor, error) {
	if !node.IsAccessible() {
		return nil, nil, ErrNodeNotFound
	}
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationRead, node.ID); err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, ErrAccessDenied
	}

	var nodes []*Node
	q := db.NewSelect().Model(&nodes).
		Where("account_id = ?", node.AccountID).
		Where("parent_id = ?", node.ID).
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL")

	dir := "ASC"
	if opts.OrderDirection == ListChildrenDirectionDesc {
		dir = "DESC"
	}

	// Apply sorting with directories always first, then ID as tiebreaker.
	//
	// Cursor-based pagination implementation notes:
	// - The cursor contains the last node from the previous page along with the sort configuration
	// - The WHERE clause uses tuple comparison (kind, field, id) to filter results after the cursor position
	// - Directories are always ordered before files (kind ASC puts 'directory' before 'file' alphabetically)
	// - ID is always sorted ASC as a tiebreaker, regardless of the main sort direction
	//
	// Why ID is always ASC:
	// - Ensures deterministic ordering when multiple items have the same sort field value
	// - Maintains consistent tiebreaker behavior across different sort directions
	// - Prevents pagination inconsistencies where items with the same name/date appear in different orders
	//   depending on whether sorting ASC or DESC
	// - The tuple comparison in the WHERE clause correctly handles the direction for the main field,
	//   while ID provides a stable secondary sort
	switch opts.OrderBy {
	case ListChildrenOrderByName:
		q = q.Order("kind ASC", "name "+dir, "id ASC")
	case ListChildrenOrderByCreatedAt:
		q = q.Order("kind ASC", "created_at "+dir, "id ASC")
	case ListChildrenOrderByUpdatedAt:
		q = q.Order("kind ASC", "updated_at "+dir, "id ASC")
	}

	// Apply cursor filter for pagination.
	// The cursor contains the last node from the previous page. We use tuple comparison
	// (kind, field, id) to find all rows that come after the cursor position in the sorted order.
	// Kind is included to handle pagination across the directory/file boundary correctly.
	// For ASC: use > to get rows after cursor
	// For DESC: use < to get rows after cursor (because "after" in descending order means lesser values)
	if opts.Cursor != nil {
		if opts.Cursor.OrderBy != opts.OrderBy {
			return nil, nil, ErrCursorMismatchedOrderField
		}
		if opts.Cursor.OrderDirection != opts.OrderDirection {
			return nil, nil, ErrCursorMismatchedDirection
		}

		var op string
		switch opts.Cursor.OrderDirection {
		case ListChildrenDirectionAsc:
			op = ">"
		case ListChildrenDirectionDesc:
			op = "<"
		}

		// Include kind in tuple comparison to handle pagination across directory/file boundary
		switch opts.Cursor.OrderBy {
		case ListChildrenOrderByName:
			q = q.Where("(kind, name, id) "+op+" (?, ?, ?)", opts.Cursor.Node.Kind, opts.Cursor.Node.Name, opts.Cursor.Node.ID)
		case ListChildrenOrderByCreatedAt:
			q = q.Where("(kind, created_at, id) "+op+" (?, ?, ?)", opts.Cursor.Node.Kind, opts.Cursor.Node.CreatedAt, opts.Cursor.Node.ID)
		case ListChildrenOrderByUpdatedAt:
			q = q.Where("(kind, updated_at, id) "+op+" (?, ?, ?)", opts.Cursor.Node.Kind, opts.Cursor.Node.UpdatedAt, opts.Cursor.Node.ID)
		}
	}

	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	} else {
		q = q.Limit(listChildrenDefaultLimit)
	}

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Node, 0), nil, nil
		}
		return nil, nil, err
	}

	if len(nodes) == 0 {
		return make([]*Node, 0), nil, nil
	}

	c := &ListChildrenCursor{
		Node:           nodes[len(nodes)-1],
		OrderBy:        opts.OrderBy,
		OrderDirection: opts.OrderDirection,
	}

	return nodes, c, nil
}

func (vfs *VirtualFS) CreateFile(ctx context.Context, db bun.IDB, opts CreateFileOptions, scope *Scope) (*Node, error) {
	if !isScopeSet(scope) {
		return nil, ErrAccessDenied
	}
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationUpload, opts.ParentID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrAccessDenied
	}

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
		AccountID: scope.AccountID,
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

	_, err = db.NewInsert().Model(&node).On("CONFLICT DO NOTHING").Returning("*").Exec(ctx)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrNodeConflict
		}
		return nil, err
	}

	return &node, nil
}

func (vfs *VirtualFS) WriteFile(ctx context.Context, db bun.IDB, node *Node, content FileContent, scope *Scope) error {
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationUpload, node.ID); err != nil {
		return err
	} else if !ok {
		return ErrAccessDenied
	}
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

	if _, err := db.NewUpdate().Model(node).
		Column(setCols...).
		WherePK().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (vfs *VirtualFS) ReadFile(ctx context.Context, db bun.IDB, node *Node, scope *Scope) (FileContent, error) {
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationRead, node.ID); err != nil {
		return EmptyFileContent(), err
	} else if !ok {
		return EmptyFileContent(), ErrAccessDenied
	}
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

func (vfs *VirtualFS) CreateDirectory(ctx context.Context, db bun.IDB, parentID uuid.UUID, name string, scope *Scope) (*Node, error) {
	if !isScopeSet(scope) {
		return nil, ErrAccessDenied
	}
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationWrite, parentID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrAccessDenied
	}

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
		AccountID: scope.AccountID,
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

func (vfs *VirtualFS) SoftDeleteNode(ctx context.Context, db bun.IDB, node *Node, scope *Scope) (*Node, error) {
	deleted, err := vfs.SoftDeleteNodes(ctx, db, []*Node{node}, scope)
	if err != nil {
		return nil, err
	}
	if len(deleted) == 0 {
		return nil, ErrNodeNotFound
	}
	return deleted[0], nil
}

func (vfs *VirtualFS) SoftDeleteNodes(ctx context.Context, db bun.IDB, nodes []*Node, scope *Scope) ([]*Node, error) {
	if !scope.Allows(OperationDelete) {
		return nil, ErrAccessDenied
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	allowed, err := vfs.filterNodesByScope(ctx, db, scope, nodes)
	if err != nil {
		return nil, err
	}
	if len(allowed) == 0 {
		return nil, ErrNodeNotFound
	}

	deletableNodes := make([]*Node, 0, len(allowed))
	nodeIDs := make([]uuid.UUID, 0, len(allowed))
	for _, node := range allowed {
		if node.IsAccessible() {
			nodeIDs = append(nodeIDs, node.ID)
			deletableNodes = append(deletableNodes, node)
		}
	}

	_, err = db.NewUpdate().Model(&deletableNodes).
		Where("id IN (?)", bun.In(nodeIDs)).
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Set("deleted_at = NOW()").
		Returning("deleted_at").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to soft delete nodes: %w", err)
	}

	return deletableNodes, nil
}

func (vfs *VirtualFS) RestoreNode(ctx context.Context, db bun.IDB, node *Node, scope *Scope) error {
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationDelete, node.ID); err != nil {
		return err
	} else if !ok {
		return ErrAccessDenied
	}
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

func (vfs *VirtualFS) RenameNode(ctx context.Context, db bun.IDB, node *Node, name string, scope *Scope) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationWrite, node.ID); err != nil {
		return err
	} else if !ok {
		return ErrAccessDenied
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

func (vfs *VirtualFS) MoveNode(ctx context.Context, db bun.IDB, node *Node, parentID uuid.UUID, scope *Scope) error {
	if !node.IsAccessible() {
		return ErrNodeNotFound
	}

	// check if the node is accessible
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationWrite, node.ID); err != nil {
		return err
	} else if !ok {
		return ErrAccessDenied
	}

	// check if the new parent is accessible
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationWrite, parentID); err != nil {
		return err
	} else if !ok {
		return ErrAccessDenied
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

// MoveNodesInSameDirectory moves multiple nodes to a new parent directory in a single operation.
// All nodes MUST have the same current parent directory; this constraint enables an
// optimization where parent paths are computed only once (2 recursive queries total)
// rather than computing full paths for each node individually (N queries).
func (vfs *VirtualFS) MoveNodesInSameDirectory(ctx context.Context, db bun.IDB, nodes []*Node, newParentID uuid.UUID, scope *Scope) (*MoveFilesResult, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationWrite, newParentID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrAccessDenied
	}

	allowedNodes, err := vfs.filterNodesByScope(ctx, db, scope, nodes)
	if err != nil {
		return nil, err
	}
	if len(allowedNodes) == 0 {
		return nil, ErrNodeNotFound
	}

	// Validate all nodes are accessible
	nodeIDs := make([]uuid.UUID, len(allowedNodes))
	nodeNames := make([]string, len(allowedNodes))
	for i, node := range allowedNodes {
		if !node.IsAccessible() {
			return nil, ErrNodeNotFound
		}
		nodeIDs[i] = node.ID
		nodeNames[i] = node.Name
	}

	var conflicts []*Node
	err = db.NewSelect().Model(&conflicts).
		Where("account_id = ?", allowedNodes[0].AccountID).
		Where("parent_id = ?", newParentID).
		Where("name IN (?)", bun.In(nodeNames)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	conflictID := make(map[uuid.UUID]struct{})
	for _, c := range conflicts {
		conflictID[c.ID] = struct{}{}
	}

	movableNodes := make([]*Node, 0, len(allowedNodes)-len(conflicts))
	for _, node := range allowedNodes {
		if _, ok := conflictID[node.ID]; !ok {
			movableNodes = append(movableNodes, node)
		}
	}

	if len(movableNodes) == 0 {
		return &MoveFilesResult{
			Conflicts: conflicts,
		}, nil
	}

	moveOps, err := vfs.keyResolver.ResolveBulkMoveOps(ctx, db, movableNodes, newParentID)
	if err != nil {
		return nil, err
	}

	_, err = db.NewUpdate().
		Model((*Node)(nil)).
		Where("id IN (?)", bun.In(nodeIDs)).
		Where("status = ?", NodeStatusReady).
		Where("deleted_at IS NULL").
		Set("parent_id = ?", newParentID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	errs := []MoveFileError{}

	for _, op := range moveOps {
		if op.OldKey != op.NewKey {
			err = vfs.blobStore.Move(ctx, op.OldKey, op.NewKey)
			if err != nil {
				if errors.Is(err, blob.ErrConflict) {
					// somehow the node is not conflicting in vfs
					// but is conflicting in the blob store
					// this is a catatrophic error, so the whole operation
					// is considered a failure
					return nil, ErrNodeConflict
				}
				errs = append(errs, MoveFileError{Node: op.Node, Error: err})
			}
		}
	}

	for _, node := range allowedNodes {
		node.ParentID = newParentID
	}

	return &MoveFilesResult{
		Moved:     movableNodes,
		Conflicts: conflicts,
		Errors:    errs,
	}, nil
}

func (vfs *VirtualFS) RealPath(ctx context.Context, db bun.IDB, node *Node, scope *Scope) (Path, error) {
	if !node.IsAccessible() {
		return nil, ErrNodeNotFound
	}
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationRead, node.ID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrAccessDenied
	}
	return buildNoteAbsolutePath(ctx, db, node)
}

func (vfs *VirtualFS) PermanentlyDeleteFiles(ctx context.Context, db bun.IDB, nodes []*Node, scope *Scope) error {
	if !scope.Allows(OperationDelete) {
		return ErrAccessDenied
	}
	if len(nodes) == 0 {
		return nil
	}

	allowed, err := vfs.filterNodesByScope(ctx, db, scope, nodes)
	if err != nil {
		return err
	}
	if len(allowed) == 0 {
		return ErrNodeNotFound
	}

	for _, n := range allowed {
		if n.Kind != NodeKindFile {
			return ErrUnsupportedOperation
		}
	}

	deletedIDs := make([]uuid.UUID, 0, len(allowed))
	for _, n := range allowed {
		err := vfs.permanentlyDeleteFileNode(ctx, db, n)
		if err != nil {
			if errors.Is(err, blob.ErrNotFound) {
				// no op if the blob does not exist
				continue
			}
			return err
		} else {
			deletedIDs = append(deletedIDs, n.ID)
		}
	}

	if len(deletedIDs) == 0 {
		return nil
	}

	_, err = db.NewDelete().Model((*Node)(nil)).
		Where("id IN (?)", bun.In(deletedIDs)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (vfs *VirtualFS) PermanentlyDeleteNode(ctx context.Context, db bun.IDB, node *Node, scope *Scope) error {
	if ok, err := vfs.canAccessNode(ctx, db, scope, OperationDelete, node.ID); err != nil {
		return err
	} else if !ok {
		return ErrAccessDenied
	}

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
		if errors.Is(err, blob.ErrNotFound) {
			// no op if the blob does not exist
			return nil
		}
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
