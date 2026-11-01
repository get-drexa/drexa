//go:build integration

package user_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/uptrace/bun"
)

func TestDBStores_SharedTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pg, err := runPostgres(ctx)
	if err != nil {
		t.Skipf("postgres testcontainer unavailable: %v", err)
	}

	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })
	postgresURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	db := database.NewFromPostgres(postgresURL)
	t.Cleanup(func() { _ = db.Close() })
	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	hashed, err := password.HashString("store-test-password")
	if err != nil {
		t.Fatal(err)
	}

	for _, commit := range []bool{true, false} {
		t.Run(fmt.Sprintf("commit=%t", commit), func(t *testing.T) {
			var u *user.User
			var org *organization.Organization
			var acc *account.Account
			var drv *drive.Drive

			rollback := errors.New("roll back registration")
			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				users := user.NewDBStore(tx)
				orgs := organization.NewDBStore(tx)
				accounts := account.NewDBStore(tx)
				drives := drive.NewDBStore(tx)
				var err error

				u, err = users.Create(ctx, user.CreateOptions{
					Email:       fmt.Sprintf("commit-%t@example.com", commit),
					DisplayName: "Store test",
					Password:    hashed,
				})
				if err != nil {
					return err
				}

				org, err = orgs.CreatePersonal(ctx, "Personal")
				if err != nil {
					return err
				}

				acc, err = accounts.Create(ctx, org.ID, u.ID, account.RoleAdmin, account.StatusActive)
				if err != nil {
					return err
				}

				drv, err = drives.Create(ctx, drive.CreateDriveOptions{
					OrgID: org.ID, OwnerAccountID: &acc.ID, Name: "My Drive", QuotaBytes: 1024,
				})
				if err != nil {
					return err
				}

				// Reads must see writes from the same uncommitted transaction.
				found, err := users.FindByEmail(ctx, u.Email)
				if err != nil {
					return err
				}

				if found.ID != u.ID {
					return fmt.Errorf("unexpected user: %s", found.ID)
				}

				if !commit {
					return rollback
				}

				return nil
			})
			if commit && err != nil || !commit && !errors.Is(err, rollback) {
				t.Fatalf("transaction: %v", err)
			}

			// Fresh stores use the database to verify all four writes committed
			// or rolled back together.
			exists, err := user.NewDBStore(db).ExistsByEmail(ctx, u.Email)
			if err != nil || exists != commit {
				t.Fatalf("user exists: got %t, want %t; error: %v", exists, commit, err)
			}

			checks := []struct {
				name       string
				find       func() error
				isNotFound func(error) bool
			}{
				{"user", func() error {
					_, err := user.NewDBStore(db).FindByID(ctx, u.ID)

					return err
				}, func(err error) bool {
					var target *user.NotFoundError

					return errors.As(err, &target)
				}},
				{"organization", func() error {
					_, err := organization.NewDBStore(db).FindByID(ctx, org.ID)

					return err
				}, func(err error) bool { return errors.Is(err, organization.ErrOrganizationNotFound) }},
				{"account", func() error {
					_, err := account.NewDBStore(db).FindByID(ctx, u.ID, acc.ID)

					return err
				}, func(err error) bool { return errors.Is(err, account.ErrAccountNotFound) }},
				{"drive", func() error {
					_, err := drive.NewDBStore(db).FindByID(ctx, drv.ID)

					return err
				}, func(err error) bool { return errors.Is(err, drive.ErrDriveNotFound) }},
			}

			for _, check := range checks {
				if err := check.find(); commit && err != nil || !commit && !check.isNotFound(err) {
					t.Errorf("%s after transaction: %v", check.name, err)
				}
			}
		})
	}
}
