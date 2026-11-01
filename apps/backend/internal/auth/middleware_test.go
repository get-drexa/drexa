package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type failingRefreshService struct {
	auth.Service
	err error
}

type countingAuthService struct {
	auth.Service
	user         *user.User
	authCalls    int
	refreshCalls int
	refreshErr   error
}

func (s *countingAuthService) AuthenticateWithAccessToken(_ context.Context, token string) (*auth.AccessTokenAuthentication, error) {
	s.authCalls++
	expires := time.Now().Add(time.Minute)
	if token == "rotated-access" {
		expires = time.Now().Add(15 * time.Minute)
	}

	return &auth.AccessTokenAuthentication{
		User: s.user,
		Claims: &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}, nil
}

func (s *countingAuthService) RefreshAccessToken(context.Context, string) (*auth.AuthenticationTokens, error) {
	s.refreshCalls++
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}

	return &auth.AuthenticationTokens{
		User:         s.user,
		AccessToken:  "rotated-access",
		RefreshToken: "rotated-refresh",
	}, nil
}

func TestMiddleware_AuthenticatesOncePerRequest(t *testing.T) {
	for _, chain := range []struct {
		name     string
		optional [2]bool
	}{
		{"required-required", [2]bool{false, false}},
		{"optional-required", [2]bool{true, false}},
		{"required-optional", [2]bool{false, true}},
		{"optional-optional", [2]bool{true, true}},
	} {
		for _, credentials := range []string{"near-expiry cookies", "refresh cookie only", "bearer"} {
			t.Run(chain.name+"/"+credentials, func(t *testing.T) {
				svc := &countingAuthService{user: &user.User{Email: "test@example.com"}}
				app := fiber.New(fiber.Config{ErrorHandler: httperr.ErrorHandler})
				router := app.Group("/api")
				for _, optional := range chain.optional {
					middleware := auth.NewAuthMiddleware(svc, auth.CookieConfig{})
					if optional {
						middleware = auth.NewOptionalAuthMiddleware(svc, auth.CookieConfig{})
					}

					router.Use(middleware)
				}

				router.Get("/protected", func(c *fiber.Ctx) error {
					if reqctx.AuthenticatedUser(c) != svc.user {
						return c.SendStatus(http.StatusUnauthorized)
					}

					return c.SendStatus(http.StatusOK)
				})

				// A second request must authenticate independently of the first.
				for request := 1; request <= 2; request++ {
					req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
					if credentials == "bearer" {
						req.Header.Set("Authorization", "Bearer original-access")
					} else {
						req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "original-refresh"})
						if credentials == "near-expiry cookies" {
							req.AddCookie(&http.Cookie{Name: "access_token", Value: "original-access"})
						}
					}

					res, err := app.Test(req)
					if err != nil {
						t.Fatal(err)
					}

					res.Body.Close()
					if res.StatusCode != http.StatusOK {
						t.Fatalf("status: got %d want 200", res.StatusCode)
					}

					if svc.authCalls != request {
						t.Fatalf("authentication calls: got %d want %d", svc.authCalls, request)
					}

					wantRefresh := request
					if credentials == "bearer" {
						wantRefresh = 0
					}

					if svc.refreshCalls != wantRefresh {
						t.Fatalf("refresh calls: got %d want %d", svc.refreshCalls, wantRefresh)
					}
				}
			})
		}
	}
}

func TestMiddleware_CachedOptionalFailureDoesNotAuthorize(t *testing.T) {
	svc := &countingAuthService{refreshErr: auth.ErrRefreshTokenReused}
	app := fiber.New(fiber.Config{ErrorHandler: httperr.ErrorHandler})
	app.Get("/", auth.NewOptionalAuthMiddleware(svc, auth.CookieConfig{}), auth.NewAuthMiddleware(svc, auth.CookieConfig{}), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "reused-refresh"})
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized || svc.refreshCalls != 1 || svc.authCalls != 0 {
		t.Fatalf("status=%d refresh calls=%d authentication calls=%d", res.StatusCode, svc.refreshCalls, svc.authCalls)
	}
}

func (s failingRefreshService) RefreshAccessToken(context.Context, string) (*auth.AuthenticationTokens, error) {
	return nil, s.err
}

func TestMiddleware_RefreshFailures(t *testing.T) {
	for _, optional := range []bool{false, true} {
		for _, tc := range []struct {
			name   string
			err    error
			status int
		}{
			{"invalid", auth.ErrInvalidRefreshToken, http.StatusUnauthorized},
			{"reused", auth.ErrRefreshTokenReused, http.StatusUnauthorized},
			{"grace collision", auth.ErrRefreshTokenRotated, http.StatusConflict},
			{"database failure", errors.New("database unavailable"), http.StatusInternalServerError},
		} {
			t.Run(tc.name+map[bool]string{true: "/optional", false: "/required"}[optional], func(t *testing.T) {
				svc := failingRefreshService{err: tc.err}
				middleware := auth.NewAuthMiddleware(svc, auth.CookieConfig{})
				expected := tc.status
				if optional {
					middleware = auth.NewOptionalAuthMiddleware(svc, auth.CookieConfig{})
					if expected == http.StatusUnauthorized {
						expected = http.StatusOK
					}
				}

				app := fiber.New(fiber.Config{ErrorHandler: httperr.ErrorHandler})
				app.Get("/", middleware, func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token"})
				response, err := app.Test(req)
				if err != nil {
					t.Fatal(err)
				}

				defer response.Body.Close()
				if response.StatusCode != expected {
					t.Fatalf("got %d want %d", response.StatusCode, expected)
				}

				if errors.Is(tc.err, auth.ErrRefreshTokenRotated) {
					if len(response.Header.Values("Set-Cookie")) != 0 {
						t.Fatal("grace collision must not set or clear cookies")
					}

					var body map[string]string
					if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}

					if body["code"] != "refresh_token_rotated" {
						t.Fatalf("expected retryable error code, got %q", body["code"])
					}
				}
			})
		}
	}
}

func TestMiddleware_AutoRefreshGracePreservesAuthentication(t *testing.T) {
	for _, optional := range []bool{false, true} {
		t.Run(map[bool]string{true: "optional", false: "required"}[optional], func(t *testing.T) {
			svc := &countingAuthService{
				user:       &user.User{Email: "test@example.com"},
				refreshErr: auth.ErrRefreshTokenRotated,
			}

			middleware := auth.NewAuthMiddleware(svc, auth.CookieConfig{})
			if optional {
				middleware = auth.NewOptionalAuthMiddleware(svc, auth.CookieConfig{})
			}

			app := fiber.New(fiber.Config{ErrorHandler: httperr.ErrorHandler})
			app.Get("/", middleware, func(c *fiber.Ctx) error {
				if reqctx.AuthenticatedUser(c) != svc.user {
					return c.SendStatus(http.StatusUnauthorized)
				}

				return c.SendStatus(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: "original-access"})
			req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "original-refresh"})
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}

			defer res.Body.Close()
			if res.StatusCode != http.StatusOK || svc.authCalls != 1 || svc.refreshCalls != 1 {
				t.Fatalf("status=%d auth calls=%d refresh calls=%d", res.StatusCode, svc.authCalls, svc.refreshCalls)
			}

			if len(res.Header.Values("Set-Cookie")) != 0 {
				t.Fatal("auto-refresh collision must not set or clear cookies")
			}
		})
	}
}
