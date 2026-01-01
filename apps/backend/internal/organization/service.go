package organization

import (
	"context"

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
		return nil, err
	}
	return &org, nil
}

