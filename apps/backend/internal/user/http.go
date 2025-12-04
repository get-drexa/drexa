package user

import (
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	service        *Service
	db             *bun.DB
	authMiddleware fiber.Handler
}

func NewHTTPHandler(service *Service, db *bun.DB, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{service: service, db: db, authMiddleware: authMiddleware}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	user := api.Group("/users")
	user.Use(h.authMiddleware)
	user.Get("/me", h.getAuthenticatedUser)
}

func (h *HTTPHandler) getAuthenticatedUser(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*User)
	if u == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	return c.JSON(u)
}
