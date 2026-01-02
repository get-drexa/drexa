//go:build integration

package account_test

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
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestService_AccountQueries(t *testing.T) {
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

	hashed, err := password.HashString("account-pass")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	userSvc := user.NewService()
	orgSvc := organization.NewService()
	accSvc := account.NewService()

	testUser, err := userSvc.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       "account@example.com",
		DisplayName: "Account User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	personalOrg, err := orgSvc.CreatePersonalOrganization(ctx, db, "Personal Org")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization(personal): %v", err)
	}

	accPersonal, err := accSvc.CreateAccount(ctx, db, personalOrg.ID, testUser.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		t.Fatalf("CreateAccount(personal): %v", err)
	}

	t.Run("list accounts", func(t *testing.T) {
		accounts, err := accSvc.ListAccounts(ctx, db, testUser.ID)
		if err != nil {
			t.Fatalf("ListAccounts: %v", err)
		}
		if len(accounts) != 1 {
			t.Fatalf("expected 1 account, got %d", len(accounts))
		}
		if accounts[0].ID != accPersonal.ID {
			t.Fatalf("unexpected account id: got %q want %q", accounts[0].ID, accPersonal.ID)
		}
	})

	t.Run("account by id", func(t *testing.T) {
		gotPersonal, err := accSvc.AccountByID(ctx, db, testUser.ID, accPersonal.ID)
		if err != nil {
			t.Fatalf("AccountByID(personal): %v", err)
		}
		if gotPersonal.OrgID != personalOrg.ID {
			t.Fatalf("unexpected personal org id: got %q want %q", gotPersonal.OrgID, personalOrg.ID)
		}
		if gotPersonal.Role != account.RoleAdmin {
			t.Fatalf("unexpected personal role: got %q want %q", gotPersonal.Role, account.RoleAdmin)
		}

	})

	t.Run("find user account in org", func(t *testing.T) {
		gotPersonal, err := accSvc.FindUserAccountInOrg(ctx, db, personalOrg.ID, testUser.ID)
		if err != nil {
			t.Fatalf("FindUserAccountInOrg(personal): %v", err)
		}
		if gotPersonal.ID != accPersonal.ID {
			t.Fatalf("unexpected personal account id: got %q want %q", gotPersonal.ID, accPersonal.ID)
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
