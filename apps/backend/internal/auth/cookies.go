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

// setAuthCookies sets HTTP-only auth cookies with security settings derived from the request.
// Secure flag is based on actual protocol (works automatically with proxies/tunnels).
func setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string, cfg CookieConfig) {
	secure := c.Protocol() == "https"

	c.Cookie(&fiber.Cookie{
		Name:     cookieKeyAccessToken,
		Value:    accessToken,
		Path:     "/",
		Domain:   cfg.Domain,
		Expires:  time.Now().Add(accessTokenValidFor),
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
		Secure:   secure,
	})
	c.Cookie(&fiber.Cookie{
		Name:     cookieKeyRefreshToken,
		Value:    refreshToken,
		Path:     "/",
		Domain:   cfg.Domain,
		Expires:  time.Now().Add(refreshTokenValidFor),
		SameSite: fiber.CookieSameSiteLaxMode,
		HTTPOnly: true,
		Secure:   secure,
	})
}
