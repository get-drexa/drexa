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

type Service struct{}

type CreateDriveOptions struct {
	OrgID          uuid.UUID
	OwnerAccountID *uuid.UUID
	Name           string
	QuotaBytes     int64
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CreateDrive(ctx context.Context, db bun.IDB, opts CreateDriveOptions) (*Drive, error) {
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

	_, err = db.NewInsert().Model(drive).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return drive, nil
}

func (s *Service) DriveByID(ctx context.Context, db bun.IDB, id uuid.UUID) (*Drive, error) {
	var drive Drive
	err := db.NewSelect().Model(&drive).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDriveNotFound
		}
		return nil, err
	}
	return &drive, nil
}

// ListAccessibleDrives returns drives a principal account can access:
// - personal drives: owner_account_id = account.ID
// - shared drives: owner_account_id IS NULL (future)
func (s *Service) ListAccessibleDrives(ctx context.Context, db bun.IDB, orgID uuid.UUID, accountID uuid.UUID) ([]*Drive, error) {
	var drives []*Drive
	err := db.NewSelect().Model(&drives).
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

// ListDrivesForUser returns all drives the user can access across orgs.
func (s *Service) ListDrivesForUser(ctx context.Context, db bun.IDB, userID uuid.UUID) ([]*Drive, error) {
	var drives []*Drive
	err := db.NewSelect().Model(&drives).
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

func (s *Service) CanAccessDrive(drive *Drive, orgID uuid.UUID, accountID uuid.UUID) bool {
	if drive == nil {
		return false
	}
	if drive.OrgID != orgID {
		return false
	}
	if drive.OwnerAccountID == nil {
		return true
	}
	return *drive.OwnerAccountID == accountID
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
