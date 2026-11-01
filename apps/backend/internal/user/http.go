package user

import (
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/gofiber/fiber/v2"
)

type HTTPHandler struct {
	authMiddleware fiber.Handler
}

func NewHTTPHandler(authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{authMiddleware: authMiddleware}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) fiber.Router {
	users := api.Group("/users")
	users.Use(h.authMiddleware)
	users.Get("/me", h.getAuthenticatedUser)

	return users
}

// getAuthenticatedUser returns the currently authenticated user
// @Summary Get current user
// @Description Retrieve the authenticated user's profile information
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} User "User profile"
// @Failure 401 {string} string "Not authenticated"
// @Router /users/me [get]
func (h *HTTPHandler) getAuthenticatedUser(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*User)
	if u == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	return c.JSON(u)
}
