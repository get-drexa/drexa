//go:build integration

package auth_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
)

func TestService_RefreshTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := runPostgres(ctx)
	if err != nil {
		t.Skipf("postgres testcontainer unavailable: %v", err)
	}

	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })
	url, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	db := database.NewFromPostgres(url)
	t.Cleanup(func() { _ = db.Close() })
	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	hashed, err := password.HashString("password123")
	if err != nil {
		t.Fatal(err)
	}

	users := user.NewDBStore(db)
	u, err := users.Create(ctx, user.CreateOptions{Email: "refresh@example.com", Password: hashed})
	if err != nil {
		t.Fatal(err)
	}

	svc := auth.NewService(db, users, auth.TokenConfig{Issuer: "test", Audience: "test", SecretKey: []byte("test-secret")})

	findToken := func(t *testing.T, token string) *auth.RefreshToken {
		t.Helper()
		data, err := base64.URLEncoding.DecodeString(token)
		if err != nil {
			t.Fatal(err)
		}

		parts, err := auth.DeserializeRefreshToken(data)
		if err != nil {
			t.Fatal(err)
		}

		rt := &auth.RefreshToken{}
		if err := db.NewSelect().Model(rt).Where("key = ?", parts.Key).Scan(ctx); err != nil {
			t.Fatal(err)
		}

		return rt
	}

	forgeSecret := func(t *testing.T, token string) string {
		t.Helper()
		data, err := base64.URLEncoding.DecodeString(token)
		if err != nil {
			t.Fatal(err)
		}

		data[len(data)-1] ^= 1
		return base64.URLEncoding.EncodeToString(data)
	}

	t.Run("failed rotation rolls back consumption", func(t *testing.T) {
		tokens, err := svc.GrantForUser(ctx, db, u)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := db.ExecContext(ctx, "ALTER TABLE refresh_tokens ADD CONSTRAINT reject_rotation CHECK (false) NOT VALID"); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			_, _ = db.ExecContext(ctx, "ALTER TABLE refresh_tokens DROP CONSTRAINT IF EXISTS reject_rotation")
		})
		result, err := svc.RefreshAccessToken(ctx, tokens.RefreshToken)
		if err == nil || result != nil {
			t.Fatal("expected failed rotation with no tokens")
		}

		if _, err := db.ExecContext(ctx, "ALTER TABLE refresh_tokens DROP CONSTRAINT reject_rotation"); err != nil {
			t.Fatal(err)
		}

		if _, err := svc.RefreshAccessToken(ctx, tokens.RefreshToken); err != nil {
			t.Fatalf("original token was consumed despite rollback: %v", err)
		}
	})

	t.Run("grace is fixed and verified reuse commits revocation", func(t *testing.T) {
		original, err := svc.GrantForUser(ctx, db, u)
		if err != nil {
			t.Fatal(err)
		}

		forged := forgeSecret(t, original.RefreshToken)
		if result, err := svc.RefreshAccessToken(ctx, forged); !errors.Is(err, auth.ErrInvalidRefreshToken) || result != nil {
			t.Fatalf("wrong secret must not rotate a token: %v", err)
		}

		rotated, err := svc.RefreshAccessToken(ctx, original.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}

		consumed := findToken(t, original.RefreshToken)
		if consumed.ConsumedAt == nil {
			t.Fatal("expected consumption timestamp")
		}

		for range 3 {
			if result, err := svc.RefreshAccessToken(ctx, original.RefreshToken); !errors.Is(err, auth.ErrRefreshTokenRotated) || result != nil {
				t.Fatalf("expected grace collision without tokens: %v", err)
			}

			current := findToken(t, original.RefreshToken)
			if current.ConsumedAt == nil || !current.ConsumedAt.Equal(*consumed.ConsumedAt) {
				t.Fatal("retry changed the original consumption timestamp")
			}
		}

		if result, err := svc.RefreshAccessToken(ctx, forged); !errors.Is(err, auth.ErrInvalidRefreshToken) || result != nil {
			t.Fatalf("wrong secret must not enter grace: %v", err)
		}

		// The successor remains usable after collisions.
		rotated, err = svc.RefreshAccessToken(ctx, rotated.RefreshToken)
		if err != nil {
			t.Fatalf("grace revoked the session: %v", err)
		}

		if _, err := db.NewUpdate().Model(consumed).Set("consumed_at = ?", time.Now().Add(-time.Minute)).WherePK().Exec(ctx); err != nil {
			t.Fatal(err)
		}

		if result, err := svc.RefreshAccessToken(ctx, forged); !errors.Is(err, auth.ErrInvalidRefreshToken) || result != nil {
			t.Fatalf("wrong secret must not revoke after grace: %v", err)
		}

		rotated, err = svc.RefreshAccessToken(ctx, rotated.RefreshToken)
		if err != nil {
			t.Fatalf("wrong secret revoked the session: %v", err)
		}

		if result, err := svc.RefreshAccessToken(ctx, original.RefreshToken); !errors.Is(err, auth.ErrRefreshTokenReused) || result != nil {
			t.Fatalf("expected reuse rejection without tokens: %v", err)
		}

		if result, err := svc.RefreshAccessToken(ctx, rotated.RefreshToken); !errors.Is(err, auth.ErrInvalidRefreshToken) || result != nil {
			t.Fatalf("rotated token survived revocation: %v", err)
		}

		revoked, err := db.NewSelect().Model((*auth.Grant)(nil)).Where("user_id = ?", u.ID).Where("revoked_at IS NOT NULL").Count(ctx)
		if err != nil || revoked != 1 {
			t.Fatalf("revocation was not committed: count=%d error=%v", revoked, err)
		}
	})

	t.Run("concurrent rotations create one successor", func(t *testing.T) {
		original, err := svc.GrantForUser(ctx, db, u)
		if err != nil {
			t.Fatal(err)
		}

		rt := findToken(t, original.RefreshToken)
		type outcome struct {
			tokens *auth.AuthenticationTokens
			err    error
		}

		const workers = 6
		start := make(chan struct{})
		results := make(chan outcome, workers)
		for range workers {
			go func() {
				<-start
				tokens, err := svc.RefreshAccessToken(ctx, original.RefreshToken)
				results <- outcome{tokens, err}
			}()
		}

		close(start)
		var winner *auth.AuthenticationTokens
		successes := 0
		for range workers {
			result := <-results
			if result.err == nil && result.tokens != nil {
				successes++
				winner = result.tokens
			} else if !errors.Is(result.err, auth.ErrRefreshTokenRotated) || result.tokens != nil {
				t.Errorf("unexpected concurrent result: tokens=%v error=%v", result.tokens != nil, result.err)
			}
		}

		if successes != 1 {
			t.Fatalf("got %d successful rotations, want 1", successes)
		}

		count, err := db.NewSelect().Model((*auth.RefreshToken)(nil)).Where("grant_id = ?", rt.GrantID).Count(ctx)
		if err != nil || count != 2 {
			t.Fatalf("expected original and one successor: count=%d error=%v", count, err)
		}

		if _, err := svc.RefreshAccessToken(ctx, winner.RefreshToken); err != nil {
			t.Fatalf("winning token is unusable: %v", err)
		}
	})
}
