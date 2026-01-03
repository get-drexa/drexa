//go:build integration

package organization_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/registration"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHTTP_OrganizationEndpoints(t *testing.T) {
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
	regSvc := registration.NewService(userSvc, orgSvc, accSvc, driveSvc, vfs)

	regResult, err := regSvc.Register(ctx, db, registration.RegisterOptions{
		Email:       "reg@example.com",
		Password:    "password123",
		DisplayName: "Reg User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	authSvc := auth.NewService(userSvc, auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})
	grant, err := authSvc.GrantForUser(ctx, db, regResult.User)
	if err != nil {
		t.Fatalf("GrantForUser: %v", err)
	}

	authMiddleware := auth.NewAuthMiddleware(authSvc, db, auth.CookieConfig{})

	app := fiber.New()
	api := app.Group("/api")
	usersAPI := user.NewHTTPHandler(userSvc, db, authMiddleware).RegisterRoutes(api)

	orgHTTP := organization.NewHTTPHandler(orgSvc, accSvc, db, authMiddleware)
	orgHTTP.RegisterUserRoutes(usersAPI)
	orgHTTP.RegisterRoutes(api)

	t.Run("list organizations (personal)", func(t *testing.T) {
		var got []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		doJSONAuth(t, app, http.MethodGet, "/api/users/me/organizations", grant.AccessToken, nil, http.StatusOK, &got)
		if len(got) != 1 {
			t.Fatalf("expected 1 organization, got %d", len(got))
		}
		if got[0].Kind != string(organization.KindPersonal) {
			t.Fatalf("unexpected org kind: %q", got[0].Kind)
		}
		if got[0].Name != "Personal" {
			t.Fatalf("unexpected org name: %q", got[0].Name)
		}
		if got[0].Slug != organization.ReservedSlug {
			t.Fatalf("unexpected personal org slug: %q", got[0].Slug)
		}
	})

	t.Run("get organization by slug (reserved personal)", func(t *testing.T) {
		var got struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		doJSONAuth(t, app, http.MethodGet, "/api/organizations/my", grant.AccessToken, nil, http.StatusOK, &got)
		if got.Kind != string(organization.KindPersonal) {
			t.Fatalf("unexpected org kind: %q", got.Kind)
		}
		if got.Name != "Personal" {
			t.Fatalf("unexpected org name: %q", got.Name)
		}
		if got.Slug != organization.ReservedSlug {
			t.Fatalf("unexpected personal org slug: %q", got.Slug)
		}
	})

	slug := "acme"
	teamOrg := &organization.Organization{
		ID:   uuid.Must(uuid.NewV7()),
		Kind: organization.KindTeam,
		Name: "Acme",
		Slug: &slug,
	}
	if _, err := db.NewInsert().Model(teamOrg).Returning("*").Exec(ctx); err != nil {
		t.Fatalf("insert team org: %v", err)
	}
	if _, err := accSvc.CreateAccount(ctx, db, teamOrg.ID, regResult.User.ID, account.RoleMember, account.StatusActive); err != nil {
		t.Fatalf("CreateAccount(team): %v", err)
	}

	t.Run("get organization by slug (team)", func(t *testing.T) {
		var got struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		doJSONAuth(t, app, http.MethodGet, "/api/organizations/acme", grant.AccessToken, nil, http.StatusOK, &got)
		if got.ID != teamOrg.ID.String() {
			t.Fatalf("unexpected org id: got %q want %q", got.ID, teamOrg.ID.String())
		}
		if got.Kind != string(organization.KindTeam) {
			t.Fatalf("unexpected org kind: %q", got.Kind)
		}
		if got.Slug != "acme" {
			t.Fatalf("unexpected org slug: %q", got.Slug)
		}
	})

	t.Run("list organizations includes team", func(t *testing.T) {
		var got []struct {
			Kind string `json:"kind"`
			Slug string `json:"slug"`
		}
		doJSONAuth(t, app, http.MethodGet, "/api/users/me/organizations", grant.AccessToken, nil, http.StatusOK, &got)
		if len(got) != 2 {
			t.Fatalf("expected 2 organizations, got %d", len(got))
		}
		seen := map[string]bool{}
		for _, o := range got {
			seen[o.Slug] = true
		}
		if !seen[organization.ReservedSlug] || !seen["acme"] {
			t.Fatalf("expected slugs my + acme, got %+v", got)
		}
	})
}

func doJSONAuth(
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
		gotBody, _ := io.ReadAll(res.Body)
		t.Fatalf("%s %s: status %d want %d body=%s", method, path, res.StatusCode, wantStatus, string(gotBody))
	}

	if out == nil {
		return
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("%s %s: decode response: %v", method, path, err)
	}
}
