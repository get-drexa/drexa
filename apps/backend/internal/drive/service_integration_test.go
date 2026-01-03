//go:build integration

package drive_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestService_DriveAccess(t *testing.T) {
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

	hashed, err := password.HashString("drive-pass")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	userSvc := user.NewService()
	orgSvc := organization.NewService()
	accSvc := account.NewService()
	driveSvc := drive.NewService()

	testUser, err := userSvc.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       "drive@example.com",
		DisplayName: "Drive User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	org1, err := orgSvc.CreatePersonalOrganization(ctx, db, "Org One")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization(org1): %v", err)
	}
	org2, err := orgSvc.CreatePersonalOrganization(ctx, db, "Org Two")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization(org2): %v", err)
	}

	acc1, err := accSvc.CreateAccount(ctx, db, org1.ID, testUser.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		t.Fatalf("CreateAccount(org1): %v", err)
	}
	acc2, err := accSvc.CreateAccount(ctx, db, org2.ID, testUser.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		t.Fatalf("CreateAccount(org2): %v", err)
	}

	drive1, err := driveSvc.CreateDrive(ctx, db, drive.CreateDriveOptions{
		OrgID:          org1.ID,
		OwnerAccountID: &acc1.ID,
		Name:           "Drive One",
		QuotaBytes:     1024 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("CreateDrive(org1): %v", err)
	}
	drive2, err := driveSvc.CreateDrive(ctx, db, drive.CreateDriveOptions{
		OrgID:          org2.ID,
		OwnerAccountID: &acc2.ID,
		Name:           "Drive Two",
		QuotaBytes:     1024 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("CreateDrive(org2): %v", err)
	}

	t.Run("list accessible drives per org", func(t *testing.T) {
		org1Drives, err := driveSvc.ListAccessibleDrives(ctx, db, org1.ID, acc1.ID)
		if err != nil {
			t.Fatalf("ListAccessibleDrives(org1): %v", err)
		}
		if len(org1Drives) != 1 {
			t.Fatalf("expected 1 drive for org1, got %d", len(org1Drives))
		}
		if org1Drives[0].ID != drive1.ID {
			t.Fatalf("unexpected org1 drive id: got %q want %q", org1Drives[0].ID, drive1.ID)
		}

		org2Drives, err := driveSvc.ListAccessibleDrives(ctx, db, org2.ID, acc2.ID)
		if err != nil {
			t.Fatalf("ListAccessibleDrives(org2): %v", err)
		}
		if len(org2Drives) != 1 {
			t.Fatalf("expected 1 drive for org2, got %d", len(org2Drives))
		}
		if org2Drives[0].ID != drive2.ID {
			t.Fatalf("unexpected org2 drive id: got %q want %q", org2Drives[0].ID, drive2.ID)
		}
	})

	t.Run("list drives for user", func(t *testing.T) {
		drives, err := driveSvc.ListDrivesForUser(ctx, db, testUser.ID)
		if err != nil {
			t.Fatalf("ListDrivesForUser: %v", err)
		}
		if len(drives) != 2 {
			t.Fatalf("expected 2 drives, got %d", len(drives))
		}
		seen := map[string]bool{
			drive1.ID.String(): false,
			drive2.ID.String(): false,
		}
		for _, d := range drives {
			if _, ok := seen[d.ID.String()]; ok {
				seen[d.ID.String()] = true
			}
		}
		for id, ok := range seen {
			if !ok {
				t.Fatalf("missing drive id %s", id)
			}
		}
	})

	t.Run("can access drive", func(t *testing.T) {
		if !driveSvc.CanAccessDrive(drive1, org1.ID, acc1.ID) {
			t.Fatalf("expected access to drive1")
		}
		if !driveSvc.CanAccessDrive(drive2, org2.ID, acc2.ID) {
			t.Fatalf("expected access to drive2")
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
