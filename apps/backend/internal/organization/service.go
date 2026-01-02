package organization

import (
	"context"
	"database/sql"
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CreatePersonalOrganization(ctx context.Context, db bun.IDB, name string) (*Organization, error) {
	id, err := newOrganizationID()
	if err != nil {
		return nil, err
	}

	org := &Organization{
		ID:   id,
		Kind: KindPersonal,
		Name: name,
	}

	_, err = db.NewInsert().Model(org).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func (s *Service) OrganizationByID(ctx context.Context, db bun.IDB, id uuid.UUID) (*Organization, error) {
	var org Organization
	err := db.NewSelect().Model(&org).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (s *Service) OrganizationBySlug(ctx context.Context, db bun.IDB, slug string) (*Organization, error) {
	var org Organization
	err := db.NewSelect().Model(&org).Where("lower(slug) = lower(?)", slug).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (s *Service) PersonalOrganizationForUser(ctx context.Context, db bun.IDB, userID uuid.UUID) (*Organization, error) {
	var org Organization
	err := db.NewSelect().Model(&org).
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
