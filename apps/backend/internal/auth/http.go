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
	tokenDeliveryCookie = "cookie"
	tokenDeliveryBody   = "body"
)

const (
	cookieKeyAccessToken  = "access_token"
	cookieKeyRefreshToken = "refresh_token"
)

type loginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	TokenDelivery string `json:"tokenDelivery"`
}

type loginResponse struct {
	User         user.User `json:"user"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
}

type refreshAccessTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
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

	case tokenDeliveryCookie:
		setAuthCookies(c, result.AccessToken, result.RefreshToken, h.cookieConfig)
		return c.JSON(loginResponse{
			User: *result.User,
		})

	case tokenDeliveryBody:
		return c.JSON(loginResponse{
			User:         *result.User,
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
		})
	}
}

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
