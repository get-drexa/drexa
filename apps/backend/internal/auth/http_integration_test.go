//go:build integration

package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestHTTP_Login_TokenDeliveryBody(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthApp(t, ctx)

	body := map[string]any{
		"email":         "reg@example.com",
		"password":      "password123",
		"tokenDelivery": auth.TokenDeliveryBody,
	}

	var resp struct {
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

	doJSON(t, app, http.MethodPost, "/api/auth/login", body, http.StatusOK, &resp)

	if resp.User.ID == "" || resp.User.Email != "reg@example.com" {
		t.Fatalf("unexpected user: %+v", resp.User)
	}
	if len(resp.Organizations) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(resp.Organizations))
	}
	if resp.Organizations[0].Kind != string(organization.KindPersonal) {
		t.Fatalf("unexpected org kind: %q", resp.Organizations[0].Kind)
	}
	if resp.Organizations[0].Slug != organization.ReservedSlug {
		t.Fatalf("unexpected personal org slug: %q", resp.Organizations[0].Slug)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected tokens in body")
	}
}

func TestHTTP_Login_TokenDeliveryCookie(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthApp(t, ctx)

	body := map[string]any{
		"email":         "reg@example.com",
		"password":      "password123",
		"tokenDelivery": auth.TokenDeliveryCookie,
	}

	res, b := doRequest(t, app, http.MethodPost, "/api/auth/login", body, http.StatusOK)

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

	setCookie := strings.Join(res.Header.Values("Set-Cookie"), "\n")
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("expected SameSite=Lax in Set-Cookie, got:\n%s", setCookie)
	}
}

func TestHTTP_Login_InvalidTokenDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthApp(t, ctx)

	body := map[string]any{
		"email":         "reg@example.com",
		"password":      "password123",
		"tokenDelivery": "nope",
	}

	doJSON(t, app, http.MethodPost, "/api/auth/login", body, http.StatusBadRequest, nil)
}

func TestHTTP_Login_InvalidCredentials(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthApp(t, ctx)

	body := map[string]any{
		"email":         "reg@example.com",
		"password":      "wrong",
		"tokenDelivery": auth.TokenDeliveryBody,
	}

	doJSON(t, app, http.MethodPost, "/api/auth/login", body, http.StatusUnauthorized, nil)
}

func TestHTTP_Tokens_Rotation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthApp(t, ctx)

	loginBody := map[string]any{
		"email":         "reg@example.com",
		"password":      "password123",
		"tokenDelivery": auth.TokenDeliveryBody,
	}

	var loginResp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	doJSON(t, app, http.MethodPost, "/api/auth/login", loginBody, http.StatusOK, &loginResp)
	if loginResp.RefreshToken == "" {
		t.Fatalf("expected refresh token from login")
	}

	var tokenResp1 struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	doJSON(t, app, http.MethodPost, "/api/auth/tokens", map[string]any{
		"refreshToken": loginResp.RefreshToken,
	}, http.StatusOK, &tokenResp1)
	if tokenResp1.RefreshToken == "" || tokenResp1.AccessToken == "" {
		t.Fatalf("expected new tokens from refresh")
	}
	if tokenResp1.RefreshToken == loginResp.RefreshToken {
		t.Fatalf("expected refresh token rotation")
	}

	doJSON(t, app, http.MethodPost, "/api/auth/tokens", map[string]any{
		"refreshToken": loginResp.RefreshToken,
	}, http.StatusUnauthorized, nil)
}

func newAuthApp(t *testing.T, ctx context.Context) *fiber.App {
	t.Helper()

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

	userSvc := user.NewService()
	orgSvc := organization.NewService()
	accSvc := account.NewService()

	hashed, err := password.HashString("password123")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	u, err := userSvc.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       "reg@example.com",
		DisplayName: "Reg User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	org, err := orgSvc.CreatePersonalOrganization(ctx, db, "Personal")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}

	if _, err := accSvc.CreateAccount(ctx, db, org.ID, u.ID, account.RoleAdmin, account.StatusActive); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	authSvc := auth.NewService(userSvc, auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})

	app := fiber.New(fiber.Config{
		ErrorHandler: httperr.ErrorHandler,
	})
	api := app.Group("/api")
	auth.NewHTTPHandler(authSvc, orgSvc, db, auth.CookieConfig{}).RegisterRoutes(api)

	return app
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
