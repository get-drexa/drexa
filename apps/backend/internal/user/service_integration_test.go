//go:build integration

package user

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestService_UserQueries(t *testing.T) {
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

	pw1, err := password.HashString("password-1")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}
	pw2, err := password.HashString("password-2")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	svc := NewService()

	user1, err := svc.RegisterUser(ctx, db, UserRegistrationOptions{
		Email:       "alice@example.com",
		DisplayName: "Alice",
		Password:    pw1,
	})
	if err != nil {
		t.Fatalf("RegisterUser(user1): %v", err)
	}
	user2, err := svc.RegisterUser(ctx, db, UserRegistrationOptions{
		Email:       "bob@example.com",
		DisplayName: "Bob",
		Password:    pw2,
	})
	if err != nil {
		t.Fatalf("RegisterUser(user2): %v", err)
	}

	t.Run("user by id", func(t *testing.T) {
		got1, err := svc.UserByID(ctx, db, user1.ID)
		if err != nil {
			t.Fatalf("UserByID(user1): %v", err)
		}
		if got1.ID != user1.ID {
			t.Fatalf("unexpected user1 id: got %q want %q", got1.ID, user1.ID)
		}
		if got1.Email != user1.Email {
			t.Fatalf("unexpected user1 email: got %q want %q", got1.Email, user1.Email)
		}

		got2, err := svc.UserByID(ctx, db, user2.ID)
		if err != nil {
			t.Fatalf("UserByID(user2): %v", err)
		}
		if got2.ID != user2.ID {
			t.Fatalf("unexpected user2 id: got %q want %q", got2.ID, user2.ID)
		}
		if got2.Email != user2.Email {
			t.Fatalf("unexpected user2 email: got %q want %q", got2.Email, user2.Email)
		}
	})

	t.Run("user by email", func(t *testing.T) {
		got1, err := svc.UserByEmail(ctx, db, user1.Email)
		if err != nil {
			t.Fatalf("UserByEmail(user1): %v", err)
		}
		if got1.ID != user1.ID {
			t.Fatalf("unexpected user1 id: got %q want %q", got1.ID, user1.ID)
		}

		got2, err := svc.UserByEmail(ctx, db, user2.Email)
		if err != nil {
			t.Fatalf("UserByEmail(user2): %v", err)
		}
		if got2.ID != user2.ID {
			t.Fatalf("unexpected user2 id: got %q want %q", got2.ID, user2.ID)
		}
	})
}

func TestService_RegisterUserConflict(t *testing.T) {
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

	pw, err := password.HashString("password-1")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	svc := NewService()

	_, err = svc.RegisterUser(ctx, db, UserRegistrationOptions{
		Email:       "conflict@example.com",
		DisplayName: "Conflict",
		Password:    pw,
	})
	if err != nil {
		t.Fatalf("RegisterUser(first): %v", err)
	}

	_, err = svc.RegisterUser(ctx, db, UserRegistrationOptions{
		Email:       "conflict@example.com",
		DisplayName: "Conflict 2",
		Password:    pw,
	})
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
	var existsErr *AlreadyExistsError
	if !errors.As(err, &existsErr) {
		t.Fatalf("expected AlreadyExistsError, got %T: %v", err, err)
	}
	if existsErr.Email != "conflict@example.com" {
		t.Fatalf("unexpected conflict email: got %q want %q", existsErr.Email, "conflict@example.com")
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
