//go:build integration

package registration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/registration"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
)

func TestHTTP_RegisterAccount_TokenDeliveryBody(t *testing.T) {
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

	authSvc := auth.NewService(userSvc, auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})

	app := fiber.New(fiber.Config{
		ErrorHandler: httperr.ErrorHandler,
	})
	api := app.Group("/api")
	registration.NewHTTPHandler(regSvc, authSvc, db, auth.CookieConfig{}).RegisterRoutes(api)

	body := map[string]any{
		"email":         "reg@example.com",
		"password":      "password123",
		"displayName":   "Reg User",
		"tokenDelivery": auth.TokenDeliveryBody,
	}

	var resp struct {
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

	doJSON(t, app, http.MethodPost, "/api/accounts", body, http.StatusOK, &resp)

	if resp.Account.ID == "" || resp.Account.OrgID == "" || resp.Account.UserID == "" {
		t.Fatalf("expected account ids to be set, got %+v", resp.Account)
	}
	if resp.User.ID == "" || resp.User.Email != "reg@example.com" || resp.User.DisplayName != "Reg User" {
		t.Fatalf("unexpected user: %+v", resp.User)
	}
	if resp.Drive.ID == "" {
		t.Fatalf("expected drive.id to be set")
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected access/refresh tokens in body")
	}
	if resp.Account.UserID != resp.User.ID {
		t.Fatalf("expected account.userId to match user.id: got %q want %q", resp.Account.UserID, resp.User.ID)
	}
}

func TestHTTP_RegisterAccount_TokenDeliveryCookie(t *testing.T) {
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

	authSvc := auth.NewService(userSvc, auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})

	app := fiber.New(fiber.Config{
		ErrorHandler: httperr.ErrorHandler,
	})
	api := app.Group("/api")
	registration.NewHTTPHandler(regSvc, authSvc, db, auth.CookieConfig{}).RegisterRoutes(api)

	body := map[string]any{
		"email":         "reg@example.com",
		"password":      "password123",
		"displayName":   "Reg User",
		"tokenDelivery": auth.TokenDeliveryCookie,
	}

	res, b := doRequest(t, app, http.MethodPost, "/api/accounts", body, http.StatusOK)

	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, string(b))
	}

	if _, ok := payload["accessToken"]; ok {
		t.Fatalf("expected accessToken to be omitted for cookie delivery")
	}
	if _, ok := payload["refreshToken"]; ok {
		t.Fatalf("expected refreshToken to be omitted for cookie delivery")
	}

	accRaw, ok := payload["account"].(map[string]any)
	if !ok {
		t.Fatalf("expected account object, got %T", payload["account"])
	}
	if accRaw["id"] == "" || accRaw["orgId"] == "" || accRaw["userId"] == "" {
		t.Fatalf("expected account ids to be set, got %+v", accRaw)
	}

	userRaw, ok := payload["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user object, got %T", payload["user"])
	}
	if userRaw["id"] == "" || userRaw["email"] != "reg@example.com" || userRaw["displayName"] != "Reg User" {
		t.Fatalf("unexpected user: %+v", userRaw)
	}

	driveRaw, ok := payload["drive"].(map[string]any)
	if !ok {
		t.Fatalf("expected drive object, got %T", payload["drive"])
	}
	if driveRaw["id"] == "" {
		t.Fatalf("expected drive.id to be set, got %+v", driveRaw)
	}

	cookies := res.Cookies()
	byName := make(map[string]*http.Cookie, len(cookies))
	for _, c := range cookies {
		byName[c.Name] = c
	}
	at, ok := byName["access_token"]
	if !ok {
		t.Fatalf("expected access_token cookie to be set (set-cookie=%v)", res.Header.Values("Set-Cookie"))
	}
	rt, ok := byName["refresh_token"]
	if !ok {
		t.Fatalf("expected refresh_token cookie to be set (set-cookie=%v)", res.Header.Values("Set-Cookie"))
	}
	if at.Value == "" || rt.Value == "" {
		t.Fatalf("expected auth cookies to have values")
	}
	if !at.HttpOnly || !rt.HttpOnly {
		t.Fatalf("expected auth cookies to be HttpOnly")
	}
	if at.Path != "/" || rt.Path != "/" {
		t.Fatalf("expected auth cookies to have Path=/ (access=%q refresh=%q)", at.Path, rt.Path)
	}

	setCookie := strings.Join(res.Header.Values("Set-Cookie"), "\n")
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("expected SameSite=Lax in Set-Cookie, got:\n%s", setCookie)
	}
}

func doRequest(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
	body any,
	wantStatus int,
) (*http.Response, []byte) {
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

	res, err := app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d want %d body=%s", method, path, res.StatusCode, wantStatus, string(b))
	}

	return res, b
}

func doJSON(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
	body any,
	wantStatus int,
	out any,
) {
	t.Helper()

	_, b := doRequest(t, app, method, path, body, wantStatus)
	if out == nil {
		return
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("%s %s: decode response: %v body=%s", method, path, err, string(b))
	}
}
