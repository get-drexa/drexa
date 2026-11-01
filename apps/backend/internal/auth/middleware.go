package auth

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
)

var errUnauthorized = errors.New("unauthorized")

type requestAuthKey struct{}

type requestAuthResult struct {
	attempt *authAttempt
	err     error
}

// authAttempt is the normalized result of parsing and (optionally) authenticating
// a request.
//
// It intentionally separates:
//   - whether the request used cookie-based auth (SetCookies == true), which enables
//     setting/refreshing HTTP-only cookies, from
//   - what credentials were available (RefreshToken), and
//   - the authenticated principal (AuthResult).
//
// This is shared by:
//   - NewAuthMiddleware (strict): rejects unauthenticated requests with 401.
//   - NewOptionalAuthMiddleware (best-effort): proceeds as anonymous on auth failure,
//     and clears auth cookies when the failure came from cookies to avoid repeated
//     failing auth attempts on public endpoints.
type authAttempt struct {
	// AuthResult is populated when access token authentication succeeds.
	AuthResult *AccessTokenAuthentication

	// SetCookies is true when the request used cookie-based auth (no Authorization header).
	// When true, the middleware may set/refresh cookies on successful auth or refresh flows.
	SetCookies bool

	// RefreshToken is the refresh token extracted from cookies when SetCookies is true.
	// It is used for auto-refreshing near-expiry access tokens.
	RefreshToken string
}

func authenticateRequest(c *fiber.Ctx, s Service, cookieConfig CookieConfig) (*authAttempt, error) {
	var at string
	var rt string
	var setCookies bool

	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return nil, errUnauthorized
		}

		at = parts[1]
		setCookies = false
	} else {
		cookies := authCookies(c)
		at = cookies[cookieKeyAccessToken]
		rt = cookies[cookieKeyRefreshToken]
		setCookies = true
	}

	if at == "" && rt == "" {
		return nil, errUnauthorized
	}

	if at == "" {
		// if there is no access token, attempt to get new access token using the refresh token.
		newTokens, err := s.RefreshAccessToken(c.Context(), rt)
		if err != nil {
			if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrRefreshTokenExpired) || errors.Is(err, ErrRefreshTokenReused) {
				return nil, errUnauthorized
			}

			return nil, err
		}

		SetAuthCookies(c, newTokens.AccessToken, newTokens.RefreshToken, cookieConfig)
		at = newTokens.AccessToken
		rt = newTokens.RefreshToken
	}

	authResult, err := s.AuthenticateWithAccessToken(c.Context(), at)
	if err != nil {
		var e *InvalidAccessTokenError

		if errors.As(err, &e) {
			return nil, errUnauthorized
		}

		var nf *user.NotFoundError

		if errors.As(err, &nf) {
			return nil, errUnauthorized
		}

		return nil, err
	}

	return &authAttempt{
		AuthResult:   authResult,
		SetCookies:   setCookies,
		RefreshToken: rt,
	}, nil
}

func maybeAutoRefresh(c *fiber.Ctx, s Service, cookieConfig CookieConfig, attempt *authAttempt) {
	if attempt == nil || attempt.AuthResult == nil {
		return
	}

	if !attempt.SetCookies || attempt.RefreshToken == "" {
		return
	}

	if time.Until(attempt.AuthResult.Claims.ExpiresAt.Time) >= 5*time.Minute {
		return
	}

	newTokens, err := s.RefreshAccessToken(c.Context(), attempt.RefreshToken)
	if errors.Is(err, ErrRefreshTokenRotated) {
		// The access token is valid. Preserve cookies from the winning request.
		return
	}

	if err != nil {
		slog.Debug("auto-refresh failed", "error", err)
		return
	}

	SetAuthCookies(c, newTokens.AccessToken, newTokens.RefreshToken, cookieConfig)
}

// authenticateOnce shares authentication and refresh across all auth middleware
// in this request. Response cookies do not replace the original request cookies,
// so authenticating again could reuse a refresh token we have already rotated.
func authenticateOnce(c *fiber.Ctx, s Service, cookieConfig CookieConfig) (*authAttempt, error) {
	if result, ok := c.Locals(requestAuthKey{}).(*requestAuthResult); ok {
		return result.attempt, result.err
	}

	attempt, err := authenticateRequest(c, s, cookieConfig)
	if err == nil {
		reqctx.SetAuthenticatedUser(c, attempt.AuthResult.User)
		maybeAutoRefresh(c, s, cookieConfig, attempt)
	}

	c.Locals(requestAuthKey{}, &requestAuthResult{attempt: attempt, err: err})

	return attempt, err
}

// NewAuthMiddleware creates a middleware that authenticates requests via Bearer token or cookies.
// To obtain the authenticated user in subsequent handlers, see reqctx.AuthenticatedUser.
func NewAuthMiddleware(s Service, cookieConfig CookieConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := authenticateOnce(c, s, cookieConfig)
		if err != nil {
			if errors.Is(err, ErrRefreshTokenRotated) {
				return refreshConflictResponse(c)
			}

			if errors.Is(err, errUnauthorized) {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			return httperr.Internal(err)
		}

		return c.Next()
	}
}

// NewOptionalAuthMiddleware attempts to authenticate a request if it contains auth
// credentials (Authorization header or auth cookies).
//
// Unlike NewAuthMiddleware, this middleware never rejects the request with 401. If
// authentication fails, it proceeds as an anonymous request. A refresh collision
// returns 409 without clearing cookies so the client can retry with its new token.
//
// This is useful for endpoints that are publicly accessible but can also behave
// differently for authenticated callers (e.g. share consumption routes that may
// resolve additional permissions for a specific account when credentials are
// present).
//
// Usage:
//
//	router.Use(auth.NewOptionalAuthMiddleware(authService, cookieConfig))
func NewOptionalAuthMiddleware(s Service, cookieConfig CookieConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		hasAuthHeader := authHeader != ""
		hasAuthCookies := c.Cookies(cookieKeyAccessToken) != "" || c.Cookies(cookieKeyRefreshToken) != ""
		if !hasAuthHeader && !hasAuthCookies {
			return c.Next()
		}

		_, err := authenticateOnce(c, s, cookieConfig)
		if err != nil {
			if errors.Is(err, ErrRefreshTokenRotated) {
				return refreshConflictResponse(c)
			}

			if errors.Is(err, errUnauthorized) {
				if !hasAuthHeader && hasAuthCookies {
					ClearAuthCookies(c, cookieConfig)
				}

				return c.Next()
			}

			return httperr.Internal(err)
		}

		return c.Next()
	}
}
