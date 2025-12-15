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
	"github.com/uptrace/bun"
)

// NewAuthMiddleware creates a middleware that authenticates requests via Bearer token or cookies.
// To obtain the authenticated user in subsequent handlers, see reqctx.AuthenticatedUser.
func NewAuthMiddleware(s *Service, db *bun.DB, cookieConfig CookieConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var at string
		var rt string
		var setCookies bool
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				slog.Info("invalid auth header")
				return c.SendStatus(fiber.StatusUnauthorized)
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
			slog.Info("no access token or refresh token")
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		if at == "" {
			// if there is no access token, attempt to get new access token using the refresh token.
			tx, err := db.BeginTx(c.Context(), nil)
			if err != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			defer tx.Rollback()

			newTokens, err := s.RefreshAccessToken(c.Context(), tx, rt)
			if err != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			if err := tx.Commit(); err != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			setAuthCookies(c, newTokens.AccessToken, newTokens.RefreshToken, cookieConfig)
			at = newTokens.AccessToken
			rt = newTokens.RefreshToken
		}

		authResult, err := s.AuthenticateWithAccessToken(c.Context(), db, at)
		if err != nil {
			var e *InvalidAccessTokenError
			if errors.As(err, &e) {
				slog.Info("invalid access token")
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			var nf *user.NotFoundError
			if errors.As(err, &nf) {
				slog.Info("user not found")
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			return httperr.Internal(err)
		}

		reqctx.SetAuthenticatedUser(c, authResult.User)

		// if cookie based auth and access token is about to expire (within 5 minutes),
		// attempt to refresh the access token. if there is any error, ignore it and let the request continue.
		if setCookies && time.Until(authResult.Claims.ExpiresAt.Time) < 5*time.Minute && rt != "" {
			tx, txErr := db.BeginTx(c.Context(), nil)
			if txErr == nil {
				newTokens, err := s.RefreshAccessToken(c.Context(), tx, rt)
				if err == nil {
					if commitErr := tx.Commit(); commitErr == nil {
						setAuthCookies(c, newTokens.AccessToken, newTokens.RefreshToken, cookieConfig)
					}
				} else {
					_ = tx.Rollback()
					slog.Debug("auto-refresh failed", "error", err)
				}
			}
		}

		return c.Next()
	}
}
