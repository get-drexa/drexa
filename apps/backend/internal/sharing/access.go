package sharing

import (
	"context"
	"errors"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var ErrItemsNotFound = errors.New("one or more items not found")

func (s *service) CreateShareForItems(ctx context.Context, driveID uuid.UUID, acc *account.Account, itemIDs []string, expiresAt time.Time, scope *virtualfs.Scope) (*Share, error) {
	if len(itemIDs) == 0 {
		return nil, ErrShareNoItems
	}

	if scope == nil || scope.DriveID != driveID || !scope.Allows(virtualfs.OperationShare) || acc == nil || acc.Status != account.StatusActive || scope.ActorKind != virtualfs.ScopeActorAccount || scope.ActorID != acc.ID {
		return nil, ErrShareNotFound
	}

	var result *Share

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		drv, err := drive.NewDBStore(tx).FindByID(ctx, driveID)
		if err != nil {
			return err
		}

		if !drive.CanAccess(drv, acc.OrgID, acc.ID) {
			return ErrShareNotFound
		}

		nodes, err := s.vfs.FindNodesByPublicID(ctx, tx, itemIDs, scope)
		if err != nil {
			return err
		}

		if len(nodes) != len(itemIDs) {
			return ErrItemsNotFound
		}

		result, err = s.createShare(ctx, tx, driveID, acc.ID, CreateShareOptions{Items: nodes, ExpiresAt: expiresAt})

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *service) FindShareForDrive(ctx context.Context, driveID uuid.UUID, publicID string) (*Share, error) {
	share, err := s.findShareByPublicID(ctx, s.db, publicID)
	if err != nil {
		return nil, err
	}

	if share.DriveID != driveID {
		return nil, ErrShareNotFound
	}

	return share, nil
}

func (s *service) manageableShare(ctx context.Context, tx bun.Tx, driveID uuid.UUID, acc *account.Account, publicID string) (*Share, error) {
	share, err := s.findShareByPublicID(ctx, tx, publicID)
	if err != nil {
		return nil, err
	}

	if share.DriveID != driveID || acc == nil || acc.Status != account.StatusActive || (acc.Role != account.RoleAdmin && share.CreatedByAccountID != acc.ID) {
		return nil, ErrShareNotFound
	}

	drv, err := drive.NewDBStore(tx).FindByID(ctx, driveID)
	if err != nil {
		return nil, err
	}

	if !drive.CanAccess(drv, acc.OrgID, acc.ID) {
		return nil, ErrShareNotFound
	}

	return share, nil
}

func (s *service) UpdateShareForAccount(ctx context.Context, driveID uuid.UUID, acc *account.Account, publicID string, opts UpdateShareOptions) (*Share, error) {
	var result *Share

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var err error

		result, err = s.manageableShare(ctx, tx, driveID, acc, publicID)
		if err != nil {
			return err
		}

		return s.updateShare(ctx, tx, result, opts)
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *service) DeleteShareForAccount(ctx context.Context, driveID uuid.UUID, acc *account.Account, publicID string) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := s.manageableShare(ctx, tx, driveID, acc, publicID); err != nil {
			return err
		}

		return s.deleteShareByPublicID(ctx, tx, publicID)
	})
}

func (s *service) ResolveAccess(ctx context.Context, publicID string, userID, orgID uuid.UUID) (*virtualfs.Scope, error) {
	share, err := s.findShareByPublicID(ctx, s.db, publicID)
	if err != nil {
		return nil, err
	}

	drv, err := drive.NewDBStore(s.db).FindByID(ctx, share.DriveID)
	if err != nil {
		return nil, err
	}

	if orgID != uuid.Nil && drv.OrgID != orgID {
		return nil, ErrShareNotFound
	}

	var consumer *account.Account

	if userID != uuid.Nil {
		consumer, err = account.NewDBStore(s.db).FindForUserInOrg(ctx, drv.OrgID, userID)
		if err != nil && !errors.Is(err, account.ErrAccountNotFound) {
			return nil, err
		}

		if consumer != nil && consumer.Status != account.StatusActive {
			consumer = nil
		}
	}

	scope, err := s.resolveScopeForShare(ctx, s.db, consumer, share)
	if errors.Is(err, ErrNoPermissions) || (err == nil && scope == nil) {
		return nil, ErrShareNotFound
	}

	return scope, err
}
