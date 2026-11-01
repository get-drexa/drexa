package organization

import (
	"context"
	"database/sql"
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/google/uuid"
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

func (s *dbStore) CreatePersonal(ctx context.Context, name string) (*Organization, error) {
	id, err := newOrganizationID()
	if err != nil {
		return nil, err
	}

	org := &Organization{
		ID:   id,
		Kind: KindPersonal,
		Name: name,
	}

	_, err = s.db.NewInsert().Model(org).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func (s *dbStore) FindByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	var org Organization

	err := s.db.NewSelect().Model(&org).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}

		return nil, err
	}

	return &org, nil
}

func (s *dbStore) FindBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization

	err := s.db.NewSelect().Model(&org).Where("lower(slug) = lower(?)", slug).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}

		return nil, err
	}

	return &org, nil
}

func (s *dbStore) FindPersonalForUser(ctx context.Context, userID uuid.UUID) (*Organization, error) {
	var org Organization

	err := s.db.NewSelect().Model(&org).
		Join("JOIN accounts ON accounts.org_id = organization.id").
		Where("accounts.user_id = ?", userID).
		Where("accounts.status = ?", account.StatusActive).
		Where("organization.kind = ?", KindPersonal).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}

		return nil, err
	}

	return &org, nil
}

func (s *dbStore) ListForUser(ctx context.Context, userID uuid.UUID) ([]Organization, error) {
	orgs := make([]Organization, 0)
	err := s.db.NewSelect().
		Model(&orgs).
		Join("JOIN accounts ON accounts.org_id = organization.id").
		Where("accounts.user_id = ?", userID).
		Where("accounts.status = ?", account.StatusActive).
		Order("organization.kind ASC", "organization.name ASC").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return orgs, nil
		}

		return nil, err
	}

	return orgs, nil
}
