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
	"github.com/golang-jwt/jwt/v5"
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
	}, http.StatusConflict, nil)

	doJSON(t, app, http.MethodPost, "/api/auth/tokens", map[string]any{
		"refreshToken": tokenResp1.RefreshToken,
	}, http.StatusOK, nil)
}

func TestHTTP_Tokens_InvalidRefreshToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthApp(t, ctx)

	// Invalid base64
	doJSON(t, app, http.MethodPost, "/api/auth/tokens", map[string]any{
		"refreshToken": "not-valid-base64!!!",
	}, http.StatusUnauthorized, nil)

	// Valid base64 but not a real token
	doJSON(t, app, http.MethodPost, "/api/auth/tokens", map[string]any{
		"refreshToken": "dGhpcyBpcyBub3QgYSByZWFsIHRva2Vu",
	}, http.StatusUnauthorized, nil)

	// Empty token
	doJSON(t, app, http.MethodPost, "/api/auth/tokens", map[string]any{
		"refreshToken": "",
	}, http.StatusUnauthorized, nil)
}

func TestHTTP_AuthMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app := newAuthAppWithProtectedRoute(t, ctx)

	t.Run("no token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		res, err := app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", res.StatusCode)
		}
	})

	t.Run("invalid bearer token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		res, err := app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", res.StatusCode)
		}
	})

	t.Run("malformed authorization header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.Header.Set("Authorization", "NotBearer token")
		res, err := app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", res.StatusCode)
		}
	})

	t.Run("valid token succeeds", func(t *testing.T) {
		// Login to get a valid token
		var loginResp struct {
			AccessToken string `json:"accessToken"`
		}

		doJSON(t, app, http.MethodPost, "/api/auth/login", map[string]any{
			"email":         "reg@example.com",
			"password":      "password123",
			"tokenDelivery": auth.TokenDeliveryBody,
		}, http.StatusOK, &loginResp)

		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		res, err := app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", res.StatusCode)
		}
	})

	t.Run("expired token returns 401", func(t *testing.T) {
		// Generate a properly signed but expired JWT using the same secret
		expiredJWT := generateExpiredJWT(t, "drexa-test", "drexa-test", []byte("drexa-test-secret"))

		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredJWT)
		res, err := app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", res.StatusCode)
		}
	})
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

	userSvc := user.NewService(user.NewDBStore(db))
	orgSvc := organization.NewService(organization.NewDBStore(db), account.NewDBStore(db))
	accSvc := account.NewService(account.NewDBStore(db))

	hashed, err := password.HashString("password123")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	u, err := userSvc.RegisterUser(ctx, user.UserRegistrationOptions{
		Email:       "reg@example.com",
		DisplayName: "Reg User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	org, err := orgSvc.CreatePersonalOrganization(ctx, "Personal")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}

	if _, err := accSvc.CreateAccount(ctx, org.ID, u.ID, account.RoleAdmin, account.StatusActive); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	authSvc := auth.NewService(db, user.NewDBStore(db), auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})

	app := fiber.New(fiber.Config{
		ErrorHandler: httperr.ErrorHandler,
	})
	api := app.Group("/api")
	auth.NewHTTPHandler(authSvc, auth.CookieConfig{}).RegisterRoutes(api)

	return app
}

func newAuthAppWithProtectedRoute(t *testing.T, ctx context.Context) *fiber.App {
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

	userSvc := user.NewService(user.NewDBStore(db))
	orgSvc := organization.NewService(organization.NewDBStore(db), account.NewDBStore(db))
	accSvc := account.NewService(account.NewDBStore(db))

	hashed, err := password.HashString("password123")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	u, err := userSvc.RegisterUser(ctx, user.UserRegistrationOptions{
		Email:       "reg@example.com",
		DisplayName: "Reg User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	org, err := orgSvc.CreatePersonalOrganization(ctx, "Personal")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}

	if _, err := accSvc.CreateAccount(ctx, org.ID, u.ID, account.RoleAdmin, account.StatusActive); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	authSvc := auth.NewService(db, user.NewDBStore(db), auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})

	authMiddleware := auth.NewAuthMiddleware(authSvc, auth.CookieConfig{})

	app := fiber.New(fiber.Config{
		ErrorHandler: httperr.ErrorHandler,
	})
	api := app.Group("/api")
	auth.NewHTTPHandler(authSvc, auth.CookieConfig{}).RegisterRoutes(api)

	// Add a protected route that requires authentication
	api.Get("/protected", authMiddleware, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

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

// generateExpiredJWT creates a properly signed JWT that expired 1 hour ago
func generateExpiredJWT(t *testing.T, issuer, audience string, secretKey []byte) string {
	t.Helper()

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{audience},
		Subject:   "test-user-id",
		ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Expired 1 hour ago
		IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)), // Issued 2 hours ago
	})

	signed, err := token.SignedString(secretKey)
	if err != nil {
		t.Fatalf("failed to sign expired JWT: %v", err)
	}

	return signed
}
