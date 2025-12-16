package auth

import (
	"errors"
	"log/slog"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

const (
	TokenDeliveryCookie = "cookie"
	TokenDeliveryBody   = "body"
)

const (
	cookieKeyAccessToken  = "access_token"
	cookieKeyRefreshToken = "refresh_token"
)

// loginRequest represents the login credentials
// @Description Login request with email, password, and token delivery preference
type loginRequest struct {
	// User's email address
	Email string `json:"email" example:"user@example.com"`
	// User's password
	Password string `json:"password" example:"secretpassword123"`
	// How to deliver tokens: "cookie" (set HTTP-only cookies) or "body" (include in response)
	TokenDelivery string `json:"tokenDelivery" example:"body" enums:"cookie,body"`
}

// loginResponse represents a successful login response
// @Description Login response containing user info and optionally tokens
type loginResponse struct {
	// Authenticated user information
	User user.User `json:"user"`
	// JWT access token (only included when tokenDelivery is "body")
	AccessToken string `json:"accessToken,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.signature"`
	// Base64 URL encoded refresh token (only included when tokenDelivery is "body")
	RefreshToken string `json:"refreshToken,omitempty" example:"dR4nD0mUu1DkZXlCeXRlc0FuZFJhbmRvbURhdGFIZXJlMTIzNDU2Nzg5MGFi"`
}

// refreshAccessTokenRequest represents a token refresh request
// @Description Request to exchange a refresh token for new tokens
type refreshAccessTokenRequest struct {
	// Base64 URL encoded refresh token
	RefreshToken string `json:"refreshToken" example:"dR4nD0mUu1DkZXlCeXRlc0FuZFJhbmRvbURhdGFIZXJlMTIzNDU2Nzg5MGFi"`
}

// tokenResponse represents new access and refresh tokens
// @Description Response containing new access token and refresh token
type tokenResponse struct {
	// New JWT access token
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.signature"`
	// New base64 URL encoded refresh token
	RefreshToken string `json:"refreshToken" example:"xK9mPqRsTuVwXyZ0AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdefgh"`
}

type HTTPHandler struct {
	service      *Service
	db           *bun.DB
	cookieConfig CookieConfig
}

func NewHTTPHandler(s *Service, db *bun.DB, cookieConfig CookieConfig) *HTTPHandler {
	return &HTTPHandler{service: s, db: db, cookieConfig: cookieConfig}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/tokens", h.refreshAccessToken)
}

// Login authenticates a user with email and password
// @Summary User login
// @Description Authenticate with email and password to receive JWT tokens. Tokens can be delivered via HTTP-only cookies or in the response body based on the tokenDelivery field.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Login credentials"
// @Success 200 {object} loginResponse "Successful authentication"
// @Failure 400 {object} map[string]string "Invalid request body or token delivery method"
// @Failure 401 {object} map[string]string "Invalid email or password"
// @Router /auth/login [post]
func (h *HTTPHandler) Login(c *fiber.Ctx) error {
	req := new(loginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	result, err := h.service.LoginWithEmailAndPassword(c.Context(), tx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
		}
		return httperr.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return httperr.Internal(err)
	}

	switch req.TokenDelivery {
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid token delivery method"})

	case TokenDeliveryCookie:
		SetAuthCookies(c, result.AccessToken, result.RefreshToken, h.cookieConfig)
		return c.JSON(loginResponse{
			User: *result.User,
		})

	case TokenDeliveryBody:
		return c.JSON(loginResponse{
			User:         *result.User,
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
		})
	}
}

// refreshAccessToken exchanges a refresh token for new access and refresh tokens
// @Summary Refresh access token
// @Description Exchange a valid refresh token for a new pair of access and refresh tokens. The old refresh token is invalidated (rotation).
// @Tags auth
// @Accept json
// @Produce json
// @Param request body refreshAccessTokenRequest true "Refresh token"
// @Success 200 {object} tokenResponse "New tokens"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Invalid, expired, or reused refresh token"
// @Router /auth/tokens [post]
func (h *HTTPHandler) refreshAccessToken(c *fiber.Ctx) error {
	req := new(refreshAccessTokenRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	result, err := h.service.RefreshAccessToken(c.Context(), tx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) ||
			errors.Is(err, ErrRefreshTokenExpired) ||
			errors.Is(err, ErrRefreshTokenReused) {
			_ = tx.Commit()
			slog.Info("invalid refresh token", "error", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
		}
		return httperr.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(tokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
