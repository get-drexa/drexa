package sharing

import (
	"errors"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	sharingService *Service
	accountService *account.Service
	vfs            *virtualfs.VirtualFS
	db             *bun.DB
	authMiddleware fiber.Handler
}

// createShareRequest represents a request to create a share link
// @Description Request to create a new share link for files or directories
type createShareRequest struct {
	// Array of file/directory IDs to share
	Items []string `json:"items" example:"mElnUNCm8F22,kRp2XYTq9A55"`
	// Optional expiration time for the share (ISO 8601)
	ExpiresAt *time.Time `json:"expiresAt" example:"2025-01-15T00:00:00Z"`
}

func NewHTTPHandler(sharingService *Service, accountService *account.Service, vfs *virtualfs.VirtualFS, db *bun.DB, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{
		sharingService: sharingService,
		accountService: accountService,
		vfs:            vfs,
		db:             db,
		authMiddleware: authMiddleware,
	}
}

func (h *HTTPHandler) RegisterShareConsumeRoutes(r fiber.Router) *virtualfs.ScopedRouter {
	g := r.Group("/shares/:shareID", h.shareMiddleware)
	return &virtualfs.ScopedRouter{Router: g}
}

func (h *HTTPHandler) RegisterShareManagementRoutes(api *account.ScopedRouter) {
	g := api.Group("/shares")
	g.Get("/:shareID", h.getShare)
	g.Post("/", h.createShare)
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

	// a share can be public or shared to specific accounts
	// if latter, the accountId query param is expected and the route should be authenticated
	// then the correct account is found using the authenticated user and the accountId query param
	// finally, the account scope is resolved for the share
	// otherwise, consumerAccount will be nil to attempt to resolve a public scope for the share

	var consumerAccount *account.Account

	qAccountID := c.Query("accountId")
	if qAccountID != "" {
		consumerAccountID, err := uuid.Parse(qAccountID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid account ID",
			})
		}

		u := reqctx.AuthenticatedUser(c).(*user.User)
		if u != nil {
			consumerAccount, err = h.accountService.AccountByID(c.Context(), h.db, u.ID, consumerAccountID)
			if err != nil {
				if errors.Is(err, account.ErrAccountNotFound) {
					return c.SendStatus(fiber.StatusNotFound)
				}
				return httperr.Internal(err)
			}

			consumerAccount, err = h.accountService.AccountByID(c.Context(), h.db, u.ID, consumerAccountID)
			if err != nil {
				if errors.Is(err, account.ErrAccountNotFound) {
					return c.SendStatus(fiber.StatusNotFound)
				}
				return httperr.Internal(err)
			}
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
// @Param accountID path string true "Account ID" format(uuid)
// @Param shareID path string true "Share ID"
// @Success 200 {object} Share "Share details"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /accounts/{accountID}/shares/{shareID} [get]
func (h *HTTPHandler) getShare(c *fiber.Ctx) error {
	shareID := c.Params("shareID")
	share, err := h.sharingService.FindShareByPublicID(c.Context(), h.db, shareID)
	if err != nil {
		return httperr.Internal(err)
	}
	return c.JSON(share)
}

// createShare creates a new share link for files or directories
// @Summary Create share
// @Description Create a new share link for one or more files or directories
// @Tags shares
// @Accept json
// @Produce json
// @Param accountID path string true "Account ID" format(uuid)
// @Param request body createShareRequest true "Share details"
// @Success 200 {object} Share "Created share"
// @Failure 400 {object} map[string]string "Invalid request or no items provided"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {object} map[string]string "One or more items not found"
// @Security BearerAuth
// @Router /accounts/{accountID}/shares [post]
func (h *HTTPHandler) createShare(c *fiber.Ctx) error {
	scope, ok := scopeFromCtx(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	acc, ok := reqctx.CurrentAccount(c).(*account.Account)
	if !ok || acc == nil {
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

	share, err := h.sharingService.CreateShare(c.Context(), tx, acc.ID, opts)
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
// @Param accountID path string true "Account ID" format(uuid)
// @Param shareID path string true "Share ID"
// @Success 204 {string} string "Share deleted"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Share not found"
// @Security BearerAuth
// @Router /accounts/{accountID}/shares/{shareID} [delete]
func (h *HTTPHandler) deleteShare(c *fiber.Ctx) error {
	shareID := c.Params("shareID")

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
