package sharing

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
	"github.com/sqids/sqids-go"
	"github.com/uptrace/bun"
)

type Service struct {
	sqid *sqids.Sqids
	vfs  *virtualfs.VirtualFS
}

type CreateShareOptions struct {
	Items     []*virtualfs.Node
	ExpiresAt time.Time
}

// ListSharesOptions is the options for ListShares.
type ListSharesOptions struct {
	// Items stores the list of nodes to filter the shares by. If nil, all shares for the account will be returned.
	Items []*virtualfs.Node

	// IncludesExpired includes shares that have expired in the results.
	IncludesExpired bool
}

type UpdateShareOptions struct {
	// ExpiresAt sets the expiration time for the share. If nil, the expiration time is not changed. If it is a zero time, the expiration time is cleared.
	ExpiresAt *time.Time
}

var (
	ErrShareNotFound = errors.New("share not found")
	ErrShareExpired  = errors.New("share expired")
	ErrShareRevoked  = errors.New("share revoked")
	ErrShareNoItems  = errors.New("share has no items")
	ErrNoPermissions = errors.New("no permissions found")
	ErrNotSameParent = errors.New("items to be shared must be in the same parent directory")
	ErrCannotShareRoot = errors.New("cannot share root directory")
)

func NewService(vfs *virtualfs.VirtualFS) (*Service, error) {
	sqid, err := sqids.New()
	if err != nil {
		return nil, err
	}
	return &Service{vfs: vfs, sqid: sqid}, nil
}

// CreateShare creates a share record for its allowed items.
// A share is a partial share of a directory: the share root is always the common parent directory of all items.
func (s *Service) CreateShare(ctx context.Context, db bun.IDB, driveID uuid.UUID, createdByAccountID uuid.UUID, opts CreateShareOptions) (*Share, error) {
	if len(opts.Items) == 0 {
		return nil, ErrShareNoItems
	}

	id, err := generateInternalID()
	if err != nil {
		return nil, err
	}

	pid, err := s.generatePublicID()
	if err != nil {
		return nil, err
	}

	sharedDirectoryID := opts.Items[0].ParentID
	if sharedDirectoryID == uuid.Nil {
		// Root directories have no parent; they are never shareable.
		return nil, ErrCannotShareRoot
	}
	for _, item := range opts.Items[1:] {
		if item.ParentID != sharedDirectoryID {
			return nil, ErrNotSameParent
		}
	}

	now := time.Now()
	sh := &Share{
		ID:                 id,
		DriveID:            driveID,
		CreatedByAccountID: createdByAccountID,
		PublicID:           pid,
		SharedDirectoryID:  sharedDirectoryID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if !opts.ExpiresAt.IsZero() {
		sh.ExpiresAt = &opts.ExpiresAt
	}

	shareItems := make([]ShareItem, len(opts.Items))
	for i, item := range opts.Items {
		si := &shareItems[i]
		si.ID, err = generateInternalID()
		if err != nil {
			return nil, err
		}
		si.ShareID = sh.ID
		si.NodeID = item.ID
		si.CreatedAt = now
		si.UpdatedAt = now
	}

	permID, err := generateInternalID()
	if err != nil {
		return nil, err
	}

	perm := SharePermission{
		ID:      permID,
		ShareID: sh.ID,
		// public access
		AccountID: nil,
		CanRead:   true,
		CanWrite:  false,
		CanDelete: false,
		CanUpload: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = db.NewInsert().Model(sh).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	_, err = db.NewInsert().Model(&shareItems).On("CONFLICT DO NOTHING").Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	_, err = db.NewInsert().Model(&perm).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return sh, nil
}

// FindShareByPublicID loads a share by its public token.
func (s *Service) FindShareByPublicID(ctx context.Context, db bun.IDB, publicID string) (*Share, error) {
	sh := new(Share)

	err := db.NewSelect().Model(sh).
		Where("public_id = ?", publicID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}

	return sh, nil
}

func (s *Service) ListShares(ctx context.Context, db bun.IDB, driveID uuid.UUID, opts ListSharesOptions) ([]Share, error) {
	var shares []Share

	q := db.NewSelect().Model(&shares).
		Where("drive_id = ?", driveID)

	if !opts.IncludesExpired {
		q = q.Where("expires_at IS NULL OR expires_at > NOW()")
	}

	if len(opts.Items) > 0 {
		ids := make([]uuid.UUID, 0, len(opts.Items))
		for _, item := range opts.Items {
			ids = append(ids, item.ID)
		}
		q = q.Where(
			"EXISTS (SELECT 1 FROM share_items si WHERE si.share_id = share.id AND si.node_id IN (?))",
			bun.In(ids),
		)
	}

	err := q.Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]Share, 0), nil
		}
		return nil, err
	}

	if len(shares) == 0 {
		return make([]Share, 0), nil
	}

	return shares, nil
}

// ResolveScopeForShare validates a share and derives a VFS scope from its permissions and items.
// share_items must contain the allowed nodes (including a directory node to share its subtree).
func (s *Service) ResolveScopeForShare(ctx context.Context, db bun.IDB, consumerAccount *account.Account, share *Share) (*virtualfs.Scope, error) {
	now := time.Now()
	if share.ExpiresAt != nil && share.ExpiresAt.Before(now) {
		return nil, ErrShareExpired
	}
	if share.RevokedAt != nil {
		return nil, ErrShareRevoked
	}

	var permList []*SharePermission

	err := db.NewSelect().Model(&permList).
		Where("share_id = ?", share.ID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}

	if len(permList) == 0 {
		return nil, ErrNoPermissions
	}

	var publicPerm *SharePermission
	var accPerm *SharePermission
	if len(permList) == 1 {
		accPerm = permList[0]
		if accPerm.AccountID == nil {
			publicPerm = accPerm
		}
	} else {
		for _, p := range permList {
			if p.AccountID != nil && consumerAccount != nil && *p.AccountID == consumerAccount.ID {
				accPerm = p
			} else if p.AccountID == nil {
				publicPerm = p
			}
		}
	}
	if accPerm == nil && publicPerm == nil {
		return nil, ErrNoPermissions
	}

	var perm *SharePermission
	if accPerm != nil {
		perm = accPerm
	} else {
		perm = publicPerm
	}

	if perm.ExpiresAt != nil && perm.ExpiresAt.Before(now) {
		return nil, ErrShareExpired
	}

	scope := &virtualfs.Scope{
		DriveID:   share.DriveID,
		RootNodeID: share.SharedDirectoryID,
	}

	if perm.AccountID == nil {
		// the share is public
		scope.ActorKind = virtualfs.ScopeActorShare
		scope.ActorID = share.ID
	} else {
		scope.ActorKind = virtualfs.ScopeActorAccount
		scope.ActorID = *perm.AccountID
	}

	scope.AllowedOps = map[virtualfs.Operation]bool{
		virtualfs.OperationRead:   perm.CanRead,
		virtualfs.OperationWrite:  perm.CanWrite,
		virtualfs.OperationDelete: perm.CanDelete,
		virtualfs.OperationUpload: perm.CanUpload,
	}

	var items []*ShareItem
	err = db.NewSelect().Model(&items).
		Where("share_id = ?", share.ID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrShareNoItems
		}
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrShareNoItems
	}

	scope.AllowedNodes = make(map[uuid.UUID]struct{})
	for _, item := range items {
		scope.AllowedNodes[item.NodeID] = struct{}{}
	}

	return scope, nil
}

// DeleteShare deletes the given share completely. It will no longer be accessible after.
func (s *Service) DeleteShareByPublicID(ctx context.Context, db bun.IDB, publicID string) error {
	r, err := db.NewDelete().Model(&Share{}).Where("public_id = ?", publicID).Exec(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrShareNotFound
		}
		return err
	}
	c, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if c == 0 {
		return ErrShareNotFound
	}
	return nil
}

func (s *Service) generatePublicID() (string, error) {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint64(b[:])
	return s.sqid.Encode([]uint64{n})
}

func (s *Service) UpdateShare(ctx context.Context, db bun.IDB, share *Share, opts UpdateShareOptions) error {
	now := time.Now()

	cols := make([]string, 0, 2)

	if opts.ExpiresAt != nil {
		newExpiresAt := opts.ExpiresAt
		if opts.ExpiresAt.IsZero() {
			newExpiresAt = nil
		}
		if !timePtrEqual(share.ExpiresAt, newExpiresAt) {
			share.ExpiresAt = newExpiresAt
			share.UpdatedAt = now
			cols = append(cols, "expires_at", "updated_at")
		}
	}

	if len(cols) > 0 {
		_, err := db.NewUpdate().Model(share).Column(cols...).WherePK().Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
