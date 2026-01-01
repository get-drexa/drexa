//go:build integration

package organization

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/database"
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

	svc := NewService()

	org, err := svc.CreatePersonalOrganization(ctx, db, "Personal Org")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}
	if org.Kind != KindPersonal {
		t.Fatalf("unexpected org kind: got %q want %q", org.Kind, KindPersonal)
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
		if got.Kind != KindPersonal {
			t.Fatalf("unexpected org kind: got %q want %q", got.Kind, KindPersonal)
		}
		if got.Name != "Personal Org" {
			t.Fatalf("unexpected org name: got %q want %q", got.Name, "Personal Org")
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
