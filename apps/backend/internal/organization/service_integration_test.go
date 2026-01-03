//go:build integration

package organization_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestService_CreatePersonalOrganization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pg, err := runPostgres(ctx)
	if err != nil {
		t.Skipf("postgres testcontainer unavailable (docker not running/configured?): %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	postgresURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	db := database.NewFromPostgres(postgresURL)
	t.Cleanup(func() { _ = db.Close() })

	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := organization.NewService()

	org, err := svc.CreatePersonalOrganization(ctx, db, "Personal Org")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}
	if org.Kind != organization.KindPersonal {
		t.Fatalf("unexpected org kind: got %q want %q", org.Kind, organization.KindPersonal)
	}
	if org.Name != "Personal Org" {
		t.Fatalf("unexpected org name: got %q want %q", org.Name, "Personal Org")
	}

	t.Run("organization by id", func(t *testing.T) {
		got, err := svc.OrganizationByID(ctx, db, org.ID)
		if err != nil {
			t.Fatalf("OrganizationByID: %v", err)
		}
		if got.ID != org.ID {
			t.Fatalf("unexpected org id: got %q want %q", got.ID, org.ID)
		}
		if got.Kind != organization.KindPersonal {
			t.Fatalf("unexpected org kind: got %q want %q", got.Kind, organization.KindPersonal)
		}
		if got.Name != "Personal Org" {
			t.Fatalf("unexpected org name: got %q want %q", got.Name, "Personal Org")
		}
	})

	t.Run("organization by slug", func(t *testing.T) {
		slug := "test-org"
		orgWithSlug := &organization.Organization{
			ID:   uuid.Must(uuid.NewV7()),
			Kind: organization.KindTeam,
			Name: "Team Org",
			Slug: &slug,
		}
		if _, err := db.NewInsert().Model(orgWithSlug).Returning("*").Exec(ctx); err != nil {
			t.Fatalf("insert org with slug: %v", err)
		}

		for _, candidate := range []string{"TEST-ORG", "Test-Org", "test-org"} {
			got, err := svc.OrganizationBySlug(ctx, db, candidate)
			if err != nil {
				t.Fatalf("OrganizationBySlug(%q): %v", candidate, err)
			}
			if got.ID != orgWithSlug.ID {
				t.Fatalf("unexpected org id: got %q want %q", got.ID, orgWithSlug.ID)
			}
			if got.Slug == nil || *got.Slug != slug {
				t.Fatalf("unexpected org slug: got %v want %q", got.Slug, slug)
			}
		}
	})

	t.Run("personal organization for user", func(t *testing.T) {
		accSvc := account.NewService()
		userSvc := user.NewService()

		hashed, err := password.HashString("test-password")
		if err != nil {
			t.Fatalf("HashString: %v", err)
		}

		u, err := userSvc.RegisterUser(ctx, db, user.UserRegistrationOptions{
			Email:       fmt.Sprintf("org-user-%s@example.com", uuid.Must(uuid.NewV7())),
			DisplayName: "Org User",
			Password:    hashed,
		})
		if err != nil {
			t.Fatalf("RegisterUser: %v", err)
		}

		acc, err := accSvc.CreateAccount(ctx, db, org.ID, u.ID, account.RoleAdmin, account.StatusActive)
		if err != nil {
			t.Fatalf("CreateAccount: %v", err)
		}

		got, err := svc.PersonalOrganizationForUser(ctx, db, acc.UserID)
		if err != nil {
			t.Fatalf("PersonalOrganizationForUser: %v", err)
		}
		if got.ID != org.ID {
			t.Fatalf("unexpected org id: got %q want %q", got.ID, org.ID)
		}
	})
}

func runPostgres(ctx context.Context) (_ *postgres.PostgresContainer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("testcontainers panic: %v", r)
		}
	}()

	return postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("drexa"),
		postgres.WithUsername("drexa"),
		postgres.WithPassword("drexa"),
		postgres.BasicWaitStrategies(),
	)
}
