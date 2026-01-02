//go:build integration

package drexa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/gofiber/fiber/v2"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRegistrationFlow(t *testing.T) {
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

	blobRoot, err := os.MkdirTemp("", "drexa-blobs-*")
	if err != nil {
		t.Fatalf("temp blob dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(blobRoot) })

	s, err := NewServer(Config{
		Server: ServerConfig{Port: 8080},
		Database: DatabaseConfig{
			PostgresURL: postgresURL,
		},
		JWT: JWTConfig{
			Issuer:    "drexa-test",
			Audience:  "drexa-test",
			SecretKey: []byte("drexa-test-secret"),
		},
		Storage: StorageConfig{
			Mode:     StorageModeFlat,
			Backend:  StorageBackendFS,
			RootPath: blobRoot,
		},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if err := database.RunMigrations(ctx, s.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	type registerResponse struct {
		Account struct {
			ID     string `json:"id"`
			OrgID  string `json:"orgId"`
			UserID string `json:"userId"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"account"`
		User struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Email       string `json:"email"`
		} `json:"user"`
		Drive struct {
			ID string `json:"id"`
		} `json:"drive"`
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}

	registerBody := map[string]any{
		"email":         "alice@example.com",
		"password":      "password123",
		"displayName":   "Alice",
		"tokenDelivery": "body",
	}

	var reg registerResponse
	doJSON(t, s.app, http.MethodPost, "/api/accounts", "", registerBody, http.StatusOK, &reg)
	if reg.AccessToken == "" {
		t.Fatalf("expected access token in registration response")
	}
	if reg.User.Email != "alice@example.com" {
		t.Fatalf("unexpected registered user email: %q", reg.User.Email)
	}
	if reg.Account.ID == "" || reg.Drive.ID == "" {
		t.Fatalf("expected account.id and drive.id to be set")
	}

	t.Run("login includes organizations", func(t *testing.T) {
		type loginResponse struct {
			User struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
				Email       string `json:"email"`
			} `json:"user"`
			Organizations []struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"organizations"`
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		}

		loginBody := map[string]any{
			"email":         "alice@example.com",
			"password":      "password123",
			"tokenDelivery": "body",
		}

		var login loginResponse
		doJSON(t, s.app, http.MethodPost, "/api/auth/login", "", loginBody, http.StatusOK, &login)
		if login.AccessToken == "" {
			t.Fatalf("expected access token in login response")
		}
		if login.User.ID != reg.User.ID {
			t.Fatalf("unexpected user id: got %q want %q", login.User.ID, reg.User.ID)
		}
		if len(login.Organizations) != 1 {
			t.Fatalf("expected 1 organization, got %d", len(login.Organizations))
		}
		if login.Organizations[0].Kind != string(organization.KindPersonal) {
			t.Fatalf("unexpected organization kind: %q", login.Organizations[0].Kind)
		}
	})

	t.Run("drives list", func(t *testing.T) {
		var drives []struct {
			ID string `json:"id"`
		}
		doJSON(t, s.app, http.MethodGet, "/api/my/drives", reg.AccessToken, nil, http.StatusOK, &drives)
		if len(drives) != 1 {
			t.Fatalf("expected 1 drive, got %d", len(drives))
		}
		if drives[0].ID != reg.Drive.ID {
			t.Fatalf("unexpected drive id: got %q want %q", drives[0].ID, reg.Drive.ID)
		}
	})

	t.Run("users/me", func(t *testing.T) {
		var me struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Email       string `json:"email"`
		}
		doJSON(t, s.app, http.MethodGet, "/api/users/me", reg.AccessToken, nil, http.StatusOK, &me)
		if me.ID != reg.User.ID {
			t.Fatalf("unexpected user id: got %q want %q", me.ID, reg.User.ID)
		}
		if me.Email != reg.User.Email {
			t.Fatalf("unexpected user email: got %q want %q", me.Email, reg.User.Email)
		}
	})

	t.Run("users/me/organizations", func(t *testing.T) {
		var orgs []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		doJSON(t, s.app, http.MethodGet, "/api/users/me/organizations", reg.AccessToken, nil, http.StatusOK, &orgs)
		if len(orgs) != 1 {
			t.Fatalf("expected 1 organization, got %d", len(orgs))
		}
		if orgs[0].Kind != string(organization.KindPersonal) {
			t.Fatalf("unexpected organization kind: %q", orgs[0].Kind)
		}
	})

	t.Run("accounts/:id", func(t *testing.T) {
		var got struct {
			ID     string `json:"id"`
			OrgID  string `json:"orgId"`
			UserID string `json:"userId"`
			Role   string `json:"role"`
			Status string `json:"status"`
		}
		doJSON(t, s.app, http.MethodGet, fmt.Sprintf("/api/accounts/%s", reg.Account.ID), reg.AccessToken, nil, http.StatusOK, &got)
		if got.ID != reg.Account.ID {
			t.Fatalf("unexpected account id: got %q want %q", got.ID, reg.Account.ID)
		}
		if got.UserID != reg.User.ID {
			t.Fatalf("unexpected account userId: got %q want %q", got.UserID, reg.User.ID)
		}
	})

	t.Run("root directory empty", func(t *testing.T) {
		var resp struct {
			Items []any `json:"items"`
		}
		doJSON(
			t,
			s.app,
			http.MethodGet,
			fmt.Sprintf("/api/my/drives/%s/directories/root/content?limit=100", reg.Drive.ID),
			reg.AccessToken,
			nil,
			http.StatusOK,
			&resp,
		)
		if len(resp.Items) != 0 {
			t.Fatalf("expected empty root directory, got %d items", len(resp.Items))
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

func doJSON(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
	accessToken string,
	body any,
	wantStatus int,
	out any,
) {
	t.Helper()

	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json marshal: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	res, err := app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()

	if res.StatusCode != wantStatus {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("%s %s: status %d want %d body=%s", method, path, res.StatusCode, wantStatus, string(b))
	}

	if out == nil {
		return
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("%s %s: decode response: %v", method, path, err)
	}
}
