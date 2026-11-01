package organization

import (
	"context"
	"errors"
	"strings"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/google/uuid"
)

type Service interface {
	ResolveForUser(ctx context.Context, slug string, userID uuid.UUID) (*Membership, error)
	CreatePersonalOrganization(ctx context.Context, name string) (*Organization, error)
	OrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	OrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	PersonalOrganizationForUser(ctx context.Context, userID uuid.UUID) (*Organization, error)
	ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]Organization, error)
}

type service struct {
	store    Store
	accounts account.Store
}

var _ Service = (*service)(nil)

func NewService(store Store, accounts account.Store) Service {
	return &service{store: store, accounts: accounts}
}

func (s *service) CreatePersonalOrganization(ctx context.Context, name string) (*Organization, error) {
	return s.store.CreatePersonal(ctx, name)
}

func (s *service) OrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.store.FindByID(ctx, id)
}

func (s *service) OrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.store.FindBySlug(ctx, slug)
}

func (s *service) PersonalOrganizationForUser(ctx context.Context, userID uuid.UUID) (*Organization, error) {
	return s.store.FindPersonalForUser(ctx, userID)
}

func (s *service) ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]Organization, error) {
	return s.store.ListForUser(ctx, userID)
}

type Membership struct {
	Organization *Organization
	Account      *account.Account
}

// ResolveForUser hides organizations from users without active membership.
func (s *service) ResolveForUser(ctx context.Context, slug string, userID uuid.UUID) (*Membership, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	var org *Organization
	var err error

	if slug == ReservedSlug {
		org, err = s.store.FindPersonalForUser(ctx, userID)
	} else {
		slug, err = NormalizeSlug(slug)
		if err != nil {
			return nil, ErrOrganizationNotFound
		}

		org, err = s.store.FindBySlug(ctx, slug)
	}

	if err != nil {
		return nil, err
	}

	acc, err := s.accounts.FindForUserInOrg(ctx, org.ID, userID)
	if errors.Is(err, account.ErrAccountNotFound) {
		return nil, ErrOrganizationNotFound
	}

	if err != nil {
		return nil, err
	}

	if acc.Status != account.StatusActive {
		return nil, ErrOrganizationNotFound
	}

	return &Membership{Organization: org, Account: acc}, nil
}
