package organization

import (
	"errors"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
)

type HTTPHandler struct {
	service        Service
	authMiddleware fiber.Handler
}

func NewHTTPHandler(service Service, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{service: service, authMiddleware: authMiddleware}
}

// RegisterUserRoutes mounts user-scoped organization routes onto an existing
// `/users` router group (which should already be protected by auth middleware).
func (h *HTTPHandler) RegisterUserRoutes(users fiber.Router) {
	users.Get("/me/organizations", h.listAuthenticatedUserOrganizations)
}

// RegisterRoutes mounts organization routes under `/organizations` and applies
// auth middleware to the group.
func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	orgs := api.Group("/organizations")
	orgs.Use(h.authMiddleware)
	orgs.Get("/:orgSlug", h.getOrganizationBySlug)
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

	orgs, err := h.service.ListOrganizationsForUser(c.Context(), u.ID)
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(orgs)
}

// getOrganizationBySlug returns an organization by slug (only if the user is a member)
// @Summary Get organization by slug
// @Description Retrieve organization information by slug (membership required)
// @Tags organizations
// @Produce json
// @Param orgSlug path string true "Organization slug"
// @Security BearerAuth
// @Success 200 {object} Organization "Organization"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Organization not found"
// @Router /organizations/{orgSlug} [get]
func (h *HTTPHandler) getOrganizationBySlug(c *fiber.Ctx) error {
	u, _ := reqctx.AuthenticatedUser(c).(*user.User)
	if u == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	membership, err := h.service.ResolveForUser(c.Context(), c.Params("orgSlug"), u.ID)
	if err != nil {
		if errors.Is(err, ErrOrganizationNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	return c.JSON(membership.Organization)
}
