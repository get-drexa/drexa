package organization

import (
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
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

func (h *HTTPHandler) RegisterRoutes(users fiber.Router) {
	users.Get("/me/organizations", h.listAuthenticatedUserOrganizations)
}

// listAuthenticatedUserOrganizations returns the organizations the current user is a member of
// @Summary List current user's organizations
// @Description Retrieve the organizations the authenticated user belongs to
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {array} Organization "Array of organizations"
// @Failure 401 {string} string "Not authenticated"
// @Router /users/me/organizations [get]
func (h *HTTPHandler) listAuthenticatedUserOrganizations(c *fiber.Ctx) error {
	u, _ := reqctx.AuthenticatedUser(c).(*user.User)
	if u == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	orgs, err := h.service.ListOrganizationsForUser(c.Context(), h.db, u.ID)
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(orgs)
}
