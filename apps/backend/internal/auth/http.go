package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

const authServiceKey = "authService"

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
	User         User   `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func RegisterAPIRoutes(api fiber.Router, s *Service) {
	auth := api.Group("/auth", func(c *fiber.Ctx) error {
		c.Locals(authServiceKey, s)
		return c.Next()
	})

	auth.Post("/login", login)
	auth.Post("/register", register)
}

func mustAuthService(c *fiber.Ctx) *Service {
	return c.Locals(authServiceKey).(*Service)
}

func login(c *fiber.Ctx) error {
	s := mustAuthService(c)

	req := new(loginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	result, err := s.LoginWithEmailAndPassword(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(loginResponse{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

func register(c *fiber.Ctx) error {
	s := mustAuthService(c)

	req := new(registerRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	result, err := s.Register(c.Context(), registerOptions{
		email:       req.Email,
		password:    req.Password,
		displayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "User already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(loginResponse{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
