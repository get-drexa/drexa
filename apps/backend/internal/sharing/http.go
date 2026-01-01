package sharing

import (
	"errors"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/nullable"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	sharingService         *Service
	accountService         *account.Service
	driveService           *drive.Service
	vfs                    *virtualfs.VirtualFS
	db                     *bun.DB
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

func NewHTTPHandler(sharingService *Service, accountService *account.Service, driveService *drive.Service, vfs *virtualfs.VirtualFS, db *bun.DB, optionalAuthMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{
		sharingService:         sharingService,
		accountService:         accountService,
		driveService:           driveService,
		vfs:                    vfs,
		db:                     db,
		optionalAuthMiddleware: optionalAuthMiddleware,
	}
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
	shareID := c.Params("shareID")

	share, err := h.sharingService.FindShareByPublicID(c.Context(), h.db, shareID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	var consumerAccount *account.Account
	u, _ := reqctx.AuthenticatedUser(c).(*user.User)
	if u != nil {
		drive, err := h.driveService.DriveByID(c.Context(), h.db, share.DriveID)
		if err != nil {
			return httperr.Internal(err)
		}

		consumerAccount, err = h.accountService.FindUserAccountInOrg(c.Context(), h.db, drive.OrgID, u.ID)
		if err != nil {
			if errors.Is(err, account.ErrAccountNotFound) {
				consumerAccount = nil
			} else {
				return httperr.Internal(err)
			}
		} else if consumerAccount.Status != account.StatusActive {
			consumerAccount = nil
		}
	}

	scope, err := h.sharingService.ResolveScopeForShare(c.Context(), h.db, consumerAccount, share)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) || errors.Is(err, ErrShareExpired) || errors.Is(err, ErrShareRevoked) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	if scope == nil {
		// no scope can be resolved for the share
		// the user is not authorized to access the share
		// return 404 to hide the existence of the share
		return c.SendStatus(fiber.StatusNotFound)
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
// @Param driveID path string true "Drive ID" format(uuid)
// @Param shareID path string true "Share ID"
// @Success 200 {object} Share "Share details"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares/{shareID} [get]
func (h *HTTPHandler) getShare(c *fiber.Ctx) error {
	shareID := c.Params("shareID")
	share, err := h.sharingService.FindShareByPublicID(c.Context(), h.db, shareID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	drive, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	if drive == nil || share.DriveID != drive.ID {
		return c.SendStatus(fiber.StatusNotFound)
	}

	return c.JSON(share)
}

// createShare creates a new share link for files or directories
// @Summary Create share
// @Description Create a new share link for one or more files or directories. All items must be in the same parent directory. Root directory cannot be shared.
// @Tags shares
// @Accept json
// @Produce json
// @Param driveID path string true "Drive ID" format(uuid)
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

	acc, ok := reqctx.CurrentAccount(c).(*account.Account)
	if !ok || acc == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	drive, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	if drive == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var req createShareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	if len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "expects at least one item to share",
		})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	nodes, err := h.vfs.FindNodesByPublicID(c.Context(), tx, req.Items, scope)
	if err != nil {
		return httperr.Internal(err)
	}
	if len(nodes) != len(req.Items) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "One or more items not found"})
	}

	opts := CreateShareOptions{
		Items: nodes,
	}
	if req.ExpiresAt != nil {
		opts.ExpiresAt = *req.ExpiresAt
	}

	share, err := h.sharingService.CreateShare(c.Context(), tx, drive.ID, acc.ID, opts)
	if err != nil {
		if errors.Is(err, ErrNotSameParent) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "items must be in the same directory"})
		}
		if errors.Is(err, ErrCannotShareRoot) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot share root directory"})
		}
		return httperr.Internal(err)
	}

	err = tx.Commit()
	if err != nil {
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
// @Param driveID path string true "Drive ID" format(uuid)
// @Param shareID path string true "Share ID"
// @Param request body patchShareRequest true "Share details"
// @Success 200 {object} Share "Updated share"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares/{shareID} [patch]
func (h *HTTPHandler) updateShare(c *fiber.Ctx) error {
	shareID := c.Params("shareID")

	share, err := h.sharingService.FindShareByPublicID(c.Context(), h.db, shareID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	drive, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	if drive == nil || share.DriveID != drive.ID {
		return c.SendStatus(fiber.StatusNotFound)
	}

	acc, _ := reqctx.CurrentAccount(c).(*account.Account)
	if acc == nil || (acc.Role != account.RoleAdmin && share.CreatedByAccountID != acc.ID) {
		return c.SendStatus(fiber.StatusNotFound)
	}

	var req patchShareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	opts := UpdateShareOptions{}
	if req.ExpiresAt.Set {
		if req.ExpiresAt.Time == nil {
			opts.ExpiresAt = &time.Time{}
		} else {
			opts.ExpiresAt = req.ExpiresAt.Time
		}
	}

	err = h.sharingService.UpdateShare(c.Context(), tx, share, opts)
	if err != nil {
		return httperr.Internal(err)
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(share)
}

// deleteShare deletes a share link
// @Summary Delete share
// @Description Delete a share link, revoking access for all users
// @Tags shares
// @Param driveID path string true "Drive ID" format(uuid)
// @Param shareID path string true "Share ID"
// @Success 204 {string} string "Share deleted"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /drives/{driveID}/shares/{shareID} [delete]
func (h *HTTPHandler) deleteShare(c *fiber.Ctx) error {
	shareID := c.Params("shareID")

	share, err := h.sharingService.FindShareByPublicID(c.Context(), h.db, shareID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	drive, _ := reqctx.CurrentDrive(c).(*drive.Drive)
	if drive == nil || share.DriveID != drive.ID {
		return c.SendStatus(fiber.StatusNotFound)
	}

	acc, _ := reqctx.CurrentAccount(c).(*account.Account)
	if acc == nil || (acc.Role != account.RoleAdmin && share.CreatedByAccountID != acc.ID) {
		return c.SendStatus(fiber.StatusNotFound)
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	err = h.sharingService.DeleteShareByPublicID(c.Context(), tx, shareID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	err = tx.Commit()
	if err != nil {
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
