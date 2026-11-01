package drive

import (
	"context"
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service interface {
	ListForUserInOrganization(ctx context.Context, orgID, userID uuid.UUID) ([]Info, error)
	ResolveAccess(ctx context.Context, publicID string, orgID uuid.UUID, acc *account.Account) (*Access, error)
	RootPublicID(ctx context.Context, scope *virtualfs.Scope) (string, error)
	CreateDrive(ctx context.Context, opts CreateDriveOptions) (*Drive, error)
	DriveByID(ctx context.Context, id uuid.UUID) (*Drive, error)
	DriveByPublicID(ctx context.Context, publicID string) (*Drive, error)
	ListAccessibleDrives(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) ([]*Drive, error)
	ListDrivesForUser(ctx context.Context, userID uuid.UUID) ([]*Drive, error)
	CanAccessDrive(drive *Drive, orgID uuid.UUID, accountID uuid.UUID) bool
}

type service struct {
	store    Store
	accounts account.Store
	db       bun.IDB
	vfs      *virtualfs.VirtualFS
}

var _ Service = (*service)(nil)

type CreateDriveOptions struct {
	OrgID          uuid.UUID
	OwnerAccountID *uuid.UUID
	Name           string
	QuotaBytes     int64
}

func NewService(store Store, accounts account.Store, db bun.IDB, vfs *virtualfs.VirtualFS) Service {
	return &service{store: store, accounts: accounts, db: db, vfs: vfs}
}

func (s *service) CreateDrive(ctx context.Context, opts CreateDriveOptions) (*Drive, error) {
	return s.store.Create(ctx, opts)
}

func (s *service) DriveByID(ctx context.Context, id uuid.UUID) (*Drive, error) {
	return s.store.FindByID(ctx, id)
}

func (s *service) DriveByPublicID(ctx context.Context, publicID string) (*Drive, error) {
	return s.store.FindByPublicID(ctx, publicID)
}

func (s *service) ListAccessibleDrives(ctx context.Context, orgID uuid.UUID, accountID uuid.UUID) ([]*Drive, error) {
	return s.store.ListAccessible(ctx, orgID, accountID)
}

func (s *service) ListDrivesForUser(ctx context.Context, userID uuid.UUID) ([]*Drive, error) {
	return s.store.ListForUser(ctx, userID)
}

func (s *service) CanAccessDrive(drive *Drive, orgID uuid.UUID, accountID uuid.UUID) bool {
	return CanAccess(drive, orgID, accountID)
}

func CanAccess(drive *Drive, orgID uuid.UUID, accountID uuid.UUID) bool {
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

// Info contains the drive and the public ID of its root directory.
type Info struct {
	Drive     *Drive
	RootDirID string
}

type Access struct {
	Info
	Scope *virtualfs.Scope
}

func (s *service) ListForUserInOrganization(ctx context.Context, orgID, userID uuid.UUID) ([]Info, error) {
	acc, err := s.accounts.FindForUserInOrg(ctx, orgID, userID)
	if errors.Is(err, account.ErrAccountNotFound) {
		return nil, ErrDriveNotFound
	}

	if err != nil {
		return nil, err
	}

	if acc.Status != account.StatusActive {
		return nil, ErrDriveNotFound
	}

	drives, err := s.store.ListAccessible(ctx, orgID, acc.ID)
	if err != nil {
		return nil, err
	}

	result := make([]Info, 0, len(drives))
	for _, drive := range drives {
		root, err := s.vfs.FindRootDirectory(ctx, s.db, drive.ID)
		if err != nil {
			return nil, err
		}

		result = append(result, Info{Drive: drive, RootDirID: root.PublicID})
	}

	return result, nil
}

func (s *service) ResolveAccess(ctx context.Context, publicID string, orgID uuid.UUID, acc *account.Account) (*Access, error) {
	drive, err := s.store.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	if acc == nil || acc.Status != account.StatusActive || acc.OrgID != orgID || !CanAccess(drive, orgID, acc.ID) {
		return nil, ErrDriveNotFound
	}

	root, err := s.vfs.FindRootDirectory(ctx, s.db, drive.ID)
	if err != nil {
		return nil, err
	}

	return &Access{
		Info: Info{Drive: drive, RootDirID: root.PublicID},
		Scope: &virtualfs.Scope{
			DriveID: drive.ID, RootNodeID: root.ID, AllowedOps: virtualfs.AllAllowedOps,
			ActorKind: virtualfs.ScopeActorAccount, ActorID: acc.ID,
		},
	}, nil
}

func (s *service) RootPublicID(ctx context.Context, scope *virtualfs.Scope) (string, error) {
	if scope == nil {
		return "", virtualfs.ErrAccessDenied
	}

	root, err := s.vfs.FindNode(ctx, s.db, scope.RootNodeID.String(), scope)
	if err != nil {
		return "", err
	}

	return root.PublicID, nil
}
