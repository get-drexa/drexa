//go:build integration

package registration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestService_Register(t *testing.T) {
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

	blobRoot, err := os.MkdirTemp("", "drexa-blobs-*")
	if err != nil {
		t.Fatalf("temp blob dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(blobRoot) })

	blobStore := blob.NewFSStore(blob.FSStoreConfig{Root: blobRoot})
	if err := blobStore.Initialize(ctx); err != nil {
		t.Fatalf("blob store init: %v", err)
	}

	vfs, err := virtualfs.New(blobStore, virtualfs.NewFlatKeyResolver())
	if err != nil {
		t.Fatalf("virtualfs.New: %v", err)
	}

	userSvc := user.NewService()
	orgSvc := organization.NewService()
	accSvc := account.NewService()
	driveSvc := drive.NewService()

	regSvc := NewService(userSvc, orgSvc, accSvc, driveSvc, vfs)

	result, err := regSvc.Register(ctx, db, RegisterOptions{
		Email:       "reg@example.com",
		Password:    "password123",
		DisplayName: "Reg User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.User == nil || result.Account == nil || result.Drive == nil {
		t.Fatalf("expected user, account, and drive to be set")
	}

	gotUser, err := userSvc.UserByEmail(ctx, db, result.User.Email)
	if err != nil {
		t.Fatalf("UserByEmail: %v", err)
	}
	if gotUser.ID != result.User.ID {
		t.Fatalf("unexpected user id: got %q want %q", gotUser.ID, result.User.ID)
	}

	gotOrg, err := orgSvc.OrganizationByID(ctx, db, result.Account.OrgID)
	if err != nil {
		t.Fatalf("OrganizationByID: %v", err)
	}
	if gotOrg.Kind != organization.KindPersonal {
		t.Fatalf("unexpected org kind: got %q want %q", gotOrg.Kind, organization.KindPersonal)
	}
	if gotOrg.Name != "Personal" {
		t.Fatalf("unexpected org name: got %q want %q", gotOrg.Name, "Personal")
	}

	gotAccount, err := accSvc.AccountByID(ctx, db, gotUser.ID, result.Account.ID)
	if err != nil {
		t.Fatalf("AccountByID: %v", err)
	}
	if gotAccount.OrgID != gotOrg.ID {
		t.Fatalf("unexpected account org: got %q want %q", gotAccount.OrgID, gotOrg.ID)
	}
	if gotAccount.UserID != gotUser.ID {
		t.Fatalf("unexpected account user: got %q want %q", gotAccount.UserID, gotUser.ID)
	}
	if gotAccount.Role != account.RoleAdmin {
		t.Fatalf("unexpected account role: got %q want %q", gotAccount.Role, account.RoleAdmin)
	}
	if gotAccount.Status != account.StatusActive {
		t.Fatalf("unexpected account status: got %q want %q", gotAccount.Status, account.StatusActive)
	}

	gotDrive, err := driveSvc.DriveByID(ctx, db, result.Drive.ID)
	if err != nil {
		t.Fatalf("DriveByID: %v", err)
	}
	if gotDrive.OrgID != gotOrg.ID {
		t.Fatalf("unexpected drive org: got %q want %q", gotDrive.OrgID, gotOrg.ID)
	}
	if gotDrive.OwnerAccountID == nil || *gotDrive.OwnerAccountID != gotAccount.ID {
		t.Fatalf("unexpected drive owner account: got %v want %v", gotDrive.OwnerAccountID, gotAccount.ID)
	}

	root, err := vfs.FindRootDirectory(ctx, db, gotDrive.ID)
	if err != nil {
		t.Fatalf("FindRootDirectory: %v", err)
	}
	if root.Name != virtualfs.RootDirectoryName {
		t.Fatalf("unexpected root name: got %q want %q", root.Name, virtualfs.RootDirectoryName)
	}
	if root.Kind != virtualfs.NodeKindDirectory {
		t.Fatalf("unexpected root kind: got %q want %q", root.Kind, virtualfs.NodeKindDirectory)
	}
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
