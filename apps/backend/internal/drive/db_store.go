package drive

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/google/uuid"
	"github.com/sqids/sqids-go"
	"github.com/uptrace/bun"
)

type dbStore struct {
	db bun.IDB
}

var _ Store = (*dbStore)(nil)

// NewDBStore uses db for every query. Pass a bun.Tx to participate in an
// existing transaction; the caller is responsible for committing or rolling back.
func NewDBStore(db bun.IDB) *dbStore {
	return &dbStore{db: db}
}

func (s *dbStore) Create(ctx context.Context, opts CreateDriveOptions) (*Drive, error) {
	publicID, err := generatePublicID()
	if err != nil {
		return nil, err
	}

	id, err := newDriveID()
	if err != nil {
		return nil, err
	}

	drive := &Drive{
		ID:                id,
		PublicID:          publicID,
		OrgID:             opts.OrgID,
		OwnerAccountID:    opts.OwnerAccountID,
		Name:              opts.Name,
		StorageQuotaBytes: opts.QuotaBytes,
	}

	_, err = s.db.NewInsert().Model(drive).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return drive, nil
}

func (s *dbStore) FindByID(ctx context.Context, id uuid.UUID) (*Drive, error) {
	var drive Drive

	err := s.db.NewSelect().Model(&drive).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDriveNotFound
		}

		return nil, err
	}

	return &drive, nil
}

func (s *dbStore) FindByPublicID(ctx context.Context, publicID string) (*Drive, error) {
	var drive Drive

	err := s.db.NewSelect().Model(&drive).Where("public_id = ?", publicID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDriveNotFound
		}

		return nil, err
	}

	return &drive, nil
}

// ListAccessible returns drives a principal account can access:
// - personal drives: owner_account_id = account.ID
// - shared drives: owner_account_id IS NULL (future)
func (s *dbStore) ListAccessible(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) ([]*Drive, error) {
	var drives []*Drive

	err := s.db.NewSelect().Model(&drives).
		Where("org_id = ?", orgID).
		Where("owner_account_id IS NULL OR owner_account_id = ?", accountID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Drive, 0), nil
		}

		return nil, err
	}

	return drives, nil
}

// ListForUser returns all drives the user can access across orgs.
func (s *dbStore) ListForUser(ctx context.Context, userID uuid.UUID) ([]*Drive, error) {
	var drives []*Drive

	err := s.db.NewSelect().Model(&drives).
		Join("JOIN accounts a ON a.org_id = drive.org_id").
		Where("a.user_id = ?", userID).
		Where("a.status = ?", account.StatusActive).
		Where("drive.owner_account_id IS NULL OR drive.owner_account_id = a.id").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]*Drive, 0), nil
		}

		return nil, err
	}

	return drives, nil
}

func generatePublicID() (string, error) {
	sqid, err := sqids.New()
	if err != nil {
		return "", err
	}

	var b [8]byte

	_, err = rand.Read(b[:])
	if err != nil {
		return "", err
	}

	n := binary.BigEndian.Uint64(b[:])

	return sqid.Encode([]uint64{n})
}
