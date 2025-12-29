package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// CookieConfig controls auth cookie behavior.
type CookieConfig struct {
	// Domain for cross-subdomain cookies (e.g., "app.com" for web.app.com + api.app.com).
	// Leave empty for same-host cookies (localhost, single domain).
	Domain string
}

// authCookies returns auth cookies from the given fiber context.
// Returns a map with the cookie names as keys and the cookie values as values.
func authCookies(c *fiber.Ctx) map[string]string {
	m := make(map[string]string)
	at := c.Cookies(cookieKeyAccessToken)
	if at != "" {
		m[cookieKeyAccessToken] = at
	}
	rt := c.Cookies(cookieKeyRefreshToken)
	if rt != "" {
		m[cookieKeyRefreshToken] = rt
	}
	return m
}

// SetAuthCookies sets HTTP-only auth cookies with security settings derived from the request.
// Secure flag is based on the request protocol (works automatically with proxies/tunnels).
func SetAuthCookies(c *fiber.Ctx, accessToken, refreshToken string, cfg CookieConfig) {
	secure := c.Protocol() == "https"

	accessTokenCookie := &fiber.Cookie{
		Name:     cookieKeyAccessToken,
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(accessTokenValidFor),
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
		Secure:   secure,
	}
	if cfg.Domain != "" {
		accessTokenCookie.Domain = cfg.Domain
	}

	refreshTokenCookie := &fiber.Cookie{
		Name:     cookieKeyRefreshToken,
		Value:    refreshToken,
		Path:     "/",
		Expires:  time.Now().Add(refreshTokenValidFor),
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
		Secure:   secure,
	}
	if cfg.Domain != "" {
		refreshTokenCookie.Domain = cfg.Domain
	}

	c.Cookie(accessTokenCookie)
	c.Cookie(refreshTokenCookie)
}

// ClearAuthCookies clears the HTTP-only auth cookies by setting them to an expired value.
func ClearAuthCookies(c *fiber.Ctx, cfg CookieConfig) {
	secure := c.Protocol() == "https"
	expired := time.Unix(0, 0)

	accessTokenCookie := &fiber.Cookie{
		Name:     cookieKeyAccessToken,
		Value:    "",
		Path:     "/",
		Expires:  expired,
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
		Secure:   secure,
	}
	if cfg.Domain != "" {
		accessTokenCookie.Domain = cfg.Domain
	}

	refreshTokenCookie := &fiber.Cookie{
		Name:     cookieKeyRefreshToken,
		Value:    "",
		Path:     "/",
		Expires:  expired,
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
		Secure:   secure,
	}
	if cfg.Domain != "" {
		refreshTokenCookie.Domain = cfg.Domain
	}

	c.Cookie(accessTokenCookie)
	c.Cookie(refreshTokenCookie)
}
