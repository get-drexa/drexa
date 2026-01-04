//go:build integration

package drexa_test

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
	"github.com/get-drexa/drexa/internal/drexa"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

	db := database.NewFromPostgres(postgresURL)
	t.Cleanup(func() { _ = db.Close() })

	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	s, err := drexa.NewServer(drexa.Config{
		Server: drexa.ServerConfig{Port: 8080},
		Database: drexa.DatabaseConfig{
			PostgresURL: postgresURL,
		},
		JWT: drexa.JWTConfig{
			Issuer:    "drexa-test",
			Audience:  "drexa-test",
			SecretKey: []byte("drexa-test-secret"),
		},
		Storage: drexa.StorageConfig{
			Mode:     drexa.StorageModeFlat,
			Backend:  drexa.StorageBackendFS,
			RootPath: blobRoot,
		},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
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
	doJSON(t, s.App(), http.MethodPost, "/api/accounts", "", registerBody, http.StatusOK, &reg)
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
		doJSON(t, s.App(), http.MethodPost, "/api/auth/login", "", loginBody, http.StatusOK, &login)
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
		if login.Organizations[0].Slug != organization.ReservedSlug {
			t.Fatalf("unexpected personal organization slug: %q", login.Organizations[0].Slug)
		}
	})

	t.Run("drives list", func(t *testing.T) {
		var drives []struct {
			ID string `json:"id"`
		}
		doJSON(t, s.App(), http.MethodGet, "/api/my/drives", reg.AccessToken, nil, http.StatusOK, &drives)
		if len(drives) != 1 {
			t.Fatalf("expected 1 drive, got %d", len(drives))
		}
		if drives[0].ID != reg.Drive.ID {
			t.Fatalf("unexpected drive id: got %q want %q", drives[0].ID, reg.Drive.ID)
		}
	})

	t.Run("drive info", func(t *testing.T) {
		var info struct {
			ID        string `json:"id"`
			RootDirID string `json:"rootDirId"`
		}
		doJSON(t, s.App(), http.MethodGet, fmt.Sprintf("/api/my/drives/%s", reg.Drive.ID), reg.AccessToken, nil, http.StatusOK, &info)
		if info.ID != reg.Drive.ID {
			t.Fatalf("unexpected drive id: got %q want %q", info.ID, reg.Drive.ID)
		}
		if info.RootDirID == "" {
			t.Fatalf("expected rootDirId to be set")
		}
	})

	t.Run("users/me", func(t *testing.T) {
		var me struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Email       string `json:"email"`
		}
		doJSON(t, s.App(), http.MethodGet, "/api/users/me", reg.AccessToken, nil, http.StatusOK, &me)
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
		doJSON(t, s.App(), http.MethodGet, "/api/users/me/organizations", reg.AccessToken, nil, http.StatusOK, &orgs)
		if len(orgs) != 1 {
			t.Fatalf("expected 1 organization, got %d", len(orgs))
		}
		if orgs[0].Kind != string(organization.KindPersonal) {
			t.Fatalf("unexpected organization kind: %q", orgs[0].Kind)
		}
		if orgs[0].Slug != organization.ReservedSlug {
			t.Fatalf("unexpected personal organization slug: %q", orgs[0].Slug)
		}
	})

	t.Run("organizations/:orgSlug", func(t *testing.T) {
		var myOrg struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		doJSON(t, s.App(), http.MethodGet, "/api/organizations/my", reg.AccessToken, nil, http.StatusOK, &myOrg)
		if myOrg.Kind != string(organization.KindPersonal) {
			t.Fatalf("unexpected personal org kind: %q", myOrg.Kind)
		}
		if myOrg.Slug != organization.ReservedSlug {
			t.Fatalf("unexpected personal org slug: %q", myOrg.Slug)
		}

		orgID, err := uuid.NewV7()
		if err != nil {
			t.Fatalf("uuid: %v", err)
		}
		accountID, err := uuid.NewV7()
		if err != nil {
			t.Fatalf("uuid: %v", err)
		}

		_, err = db.ExecContext(ctx,
			`INSERT INTO organizations (id, kind, name, slug) VALUES (?, ?, ?, ?)`,
			orgID, organization.KindTeam, "Acme", "acme",
		)
		if err != nil {
			t.Fatalf("insert org: %v", err)
		}

		_, err = db.ExecContext(ctx,
			`INSERT INTO accounts (id, org_id, user_id, role, status) VALUES (?, ?, ?, ?, ?)`,
			accountID, orgID, reg.User.ID, "member", "active",
		)
		if err != nil {
			t.Fatalf("insert account: %v", err)
		}

		var got struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		doJSON(t, s.App(), http.MethodGet, "/api/organizations/acme", reg.AccessToken, nil, http.StatusOK, &got)
		if got.Slug != "acme" {
			t.Fatalf("unexpected org slug: %q", got.Slug)
		}
		if got.Kind != string(organization.KindTeam) {
			t.Fatalf("unexpected org kind: %q", got.Kind)
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
		doJSON(t, s.App(), http.MethodGet, fmt.Sprintf("/api/accounts/%s", reg.Account.ID), reg.AccessToken, nil, http.StatusOK, &got)
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
			s.App(),
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
