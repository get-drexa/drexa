package sharing

import (
	"errors"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/nullable"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type HTTPHandler struct {
	sharingService         Service
	optionalAuthMiddleware fiber.Handler
}

// createShareRequest represents a request to create a share link
// @Description Request to create a new share link for files or directories
type createShareRequest struct {
	// Array of file/directory IDs to share
	Items []string `json:"items" example:"mElnUNCm8F22,kRp2XYTq9A55"`
	// Optional expiration time for the share (ISO 8601)
	ExpiresAt *time.Time `json:"expiresAt" example:"2025-01-15T00:00:00Z"`
}

// patchShareRequest represents a request to update a share link
// @Description Request to update a share link. Omit expiresAt to keep the current value. Use null to remove the expiry.
type patchShareRequest struct {
	// Optional expiration time for the share (ISO 8601), null clears it.
	ExpiresAt nullable.Time `json:"expiresAt" example:"2025-01-15T00:00:00Z"`
}

func NewHTTPHandler(service Service, optionalAuthMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{sharingService: service, optionalAuthMiddleware: optionalAuthMiddleware}
}

func (h *HTTPHandler) RegisterShareConsumeRoutes(r fiber.Router) *virtualfs.ScopedRouter {
	// Public shares should be accessible without authentication. However, if the client provides auth
	// credentials (cookies or Authorization header), attempt auth so share scopes can be resolved for
	// account-scoped shares.
	g := r.Group("/shares/:shareID", h.optionalAuthMiddleware, h.shareMiddleware)

	return &virtualfs.ScopedRouter{Router: g}
}

func (h *HTTPHandler) RegisterShareManagementRoutes(api *virtualfs.ScopedRouter) {
	g := api.Group("/shares")
	g.Post("/", h.createShare)
	g.Get("/:shareID", h.getShare)
	g.Patch("/:shareID", h.updateShare)
	g.Delete("/:shareID", h.deleteShare)
}

func (h *HTTPHandler) shareMiddleware(c *fiber.Ctx) error {

	var userID, orgID uuid.UUID

	if u, _ := reqctx.AuthenticatedUser(c).(*user.User); u != nil {
		userID = u.ID
	}

	if org, _ := reqctx.CurrentOrganization(c).(*organization.Organization); org != nil {
		orgID = org.ID
	}

	scope, err := h.sharingService.ResolveAccess(c.Context(), c.Params("shareID"), userID, orgID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) || errors.Is(err, ErrShareExpired) || errors.Is(err, ErrShareRevoked) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	reqctx.SetVFSAccessScope(c, scope)

	return c.Next()
}

// getShare retrieves a share by its ID
// @Summary Get share
// @Description Retrieve share link details by ID
// @Tags shares
// @Accept json
// @Produce json
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param shareID path string true "Share ID"
// @Success 200 {object} Share "Share details"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares/{shareID} [get]
func (h *HTTPHandler) getShare(c *fiber.Ctx) error {

	drv, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	if drv == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	share, err := h.sharingService.FindShareForDrive(c.Context(), drv.ID, c.Params("shareID"))
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	return c.JSON(share)
}

// createShare creates a new share link for files or directories
// @Summary Create share
// @Description Create a new share link for one or more files or directories. All items must be in the same parent directory. Root directory cannot be shared.
// @Tags shares
// @Accept json
// @Produce json
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param request body createShareRequest true "Share details"
// @Success 200 {object} Share "Created share"
// @Failure 400 {object} map[string]string "Invalid request, items not in same directory, or root directory cannot be shared"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {object} map[string]string "One or more items not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares [post]
func (h *HTTPHandler) createShare(c *fiber.Ctx) error {

	scope, ok := scopeFromCtx(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	acc, _ := reqctx.CurrentAccount(c).(*account.Account)
	drv, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	if acc == nil || drv == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var req createShareRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	var expiresAt time.Time

	if req.ExpiresAt != nil {
		expiresAt = *req.ExpiresAt
	}

	share, err := h.sharingService.CreateShareForItems(c.Context(), drv.ID, acc, req.Items, expiresAt, scope)
	if err != nil {
		switch {
		case errors.Is(err, ErrShareNoItems):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "expects at least one item to share"})
		case errors.Is(err, ErrItemsNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "One or more items not found"})
		case errors.Is(err, ErrShareNotFound), errors.Is(err, virtualfs.ErrAccessDenied):
			return c.SendStatus(fiber.StatusNotFound)
		case errors.Is(err, ErrNotSameParent):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "items must be in the same directory"})
		case errors.Is(err, ErrCannotShareRoot):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot share root directory"})
		}

		return httperr.Internal(err)
	}

	return c.JSON(share)
}

// updateShare updates a share link
// @Summary Update share
// @Description Update share link details. Omit expiresAt to keep the current value. Use null to remove the expiry.
// @Tags shares
// @Accept json
// @Produce json
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param shareID path string true "Share ID"
// @Param request body patchShareRequest true "Share details"
// @Success 200 {object} Share "Updated share"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares/{shareID} [patch]
func (h *HTTPHandler) updateShare(c *fiber.Ctx) error {

	drv, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	acc, _ := reqctx.CurrentAccount(c).(*account.Account)
	if drv == nil || acc == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	var req patchShareRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	opts := UpdateShareOptions{}
	if req.ExpiresAt.Set {
		if req.ExpiresAt.Time == nil {
			opts.ExpiresAt = &time.Time{}
		} else {
			opts.ExpiresAt = req.ExpiresAt.Time
		}
	}

	share, err := h.sharingService.UpdateShareForAccount(c.Context(), drv.ID, acc, c.Params("shareID"), opts)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	return c.JSON(share)
}

// deleteShare deletes a share link
// @Summary Delete share
// @Description Delete a share link, revoking access for all users
// @Tags shares
// @Param driveID path string true "Drive ID" example:"kRp2XYTq9A55"
// @Param shareID path string true "Share ID"
// @Success 204 {string} string "Share deleted"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares/{shareID} [delete]
func (h *HTTPHandler) deleteShare(c *fiber.Ctx) error {

	drv, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	acc, _ := reqctx.CurrentAccount(c).(*account.Account)
	if drv == nil || acc == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	err := h.sharingService.DeleteShareForAccount(c.Context(), drv.ID, acc, c.Params("shareID"))
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func scopeFromCtx(c *fiber.Ctx) (*virtualfs.Scope, bool) {
	scopeAny := reqctx.VFSAccessScope(c)
	if scopeAny == nil {
		return nil, false
	}

	scope, ok := scopeAny.(*virtualfs.Scope)
	if !ok || scope == nil {
		return nil, false
	}

	return scope, true
}
