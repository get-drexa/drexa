package auth

import (
	"errors"
	"log/slog"

	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginResponse struct {
	User         user.User `json:"user"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
}

type HTTPHandler struct {
	service *Service
	db      *bun.DB
}

func NewHTTPHandler(s *Service, db *bun.DB) *HTTPHandler {
	return &HTTPHandler{service: s, db: db}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")

	auth.Post("/login", h.Login)
	auth.Post("/register", h.Register)
}

func (h *HTTPHandler) Login(c *fiber.Ctx) error {
	req := new(loginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}
	defer tx.Rollback()

	result, err := h.service.LoginWithEmailAndPassword(c.Context(), tx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(loginResponse{
		User:         *result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

func (h *HTTPHandler) Register(c *fiber.Ctx) error {
	req := new(registerRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}
	defer tx.Rollback()

	result, err := h.service.Register(c.Context(), tx, registerOptions{
		email:       req.Email,
		password:    req.Password,
		displayName: req.DisplayName,
	})
	if err != nil {
		var ae *user.AlreadyExistsError
		if errors.As(err, &ae) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "User already exists"})
		}
		slog.Error("failed to register user", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(loginResponse{
		User:         *result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
