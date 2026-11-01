package drive

import (
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
)

type driveInfoResponse struct {
	*Drive
	RootDirID string `json:"rootDirId"`
}

type driveListItemResponse struct {
	*Drive
	RootDir string `json:"rootDir"`
}

type HTTPHandler struct {
	driveService   Service
	authMiddleware fiber.Handler
}

func NewHTTPHandler(driveService Service, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{
		driveService:   driveService,
		authMiddleware: authMiddleware,
	}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) *virtualfs.ScopedRouter {
	api.Get("/drives", h.authMiddleware, h.listDrives)

	drive := api.Group("/drives/:driveID")
	drive.Use(h.authMiddleware)
	drive.Use(h.driveMiddleware)

	drive.Get("/", h.getDriveInfo)

	return &virtualfs.ScopedRouter{Router: drive}
}

// listDrives returns accessible drives with their root directory public IDs.
// @Summary List accessible drives
// @Tags drives
// @Produce json
// @Param orgSlug path string true "Organization slug"
// @Security BearerAuth
// @Success 200 {array} driveListItemResponse
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Organization not found"
// @Router /{orgSlug}/drives [get]
func (h *HTTPHandler) listDrives(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*user.User)
	org, ok := reqctx.CurrentOrganization(c).(*organization.Organization)
	if !ok || org == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	drives, err := h.driveService.ListForUserInOrganization(c.Context(), org.ID, u.ID)
	if err != nil {
		if errors.Is(err, ErrDriveNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	response := make([]driveListItemResponse, 0, len(drives))
	for _, info := range drives {
		response = append(response, driveListItemResponse{Drive: info.Drive, RootDir: info.RootDirID})
	}

	return c.JSON(response)
}

func (h *HTTPHandler) getDriveInfo(c *fiber.Ctx) error {
	drive, ok := reqctx.CurrentDrive(c).(*Drive)
	if !ok || drive == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	rootDirID, _ := c.Locals("rootDirId").(string)
	if rootDirID == "" {
		scopeAny := reqctx.VFSAccessScope(c)
		scope, ok := scopeAny.(*virtualfs.Scope)
		if !ok || scope == nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		rootID, err := h.driveService.RootPublicID(c.Context(), scope)
		if err != nil {
			if errors.Is(err, virtualfs.ErrNodeNotFound) || errors.Is(err, virtualfs.ErrAccessDenied) {
				return c.SendStatus(fiber.StatusNotFound)
			}

			return httperr.Internal(err)
		}

		rootDirID = rootID
	}

	return c.JSON(driveInfoResponse{
		Drive:     drive,
		RootDirID: rootDirID,
	})
}

func (h *HTTPHandler) driveMiddleware(c *fiber.Ctx) error {
	org, ok := reqctx.CurrentOrganization(c).(*organization.Organization)
	if !ok || org == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	acc, _ := reqctx.CurrentAccount(c).(*account.Account)
	access, err := h.driveService.ResolveAccess(c.Context(), c.Params("driveID"), org.ID, acc)
	if err != nil {
		if errors.Is(err, ErrDriveNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return httperr.Internal(err)
	}

	c.Locals("rootDirId", access.RootDirID)
	reqctx.SetCurrentDrive(c, access.Drive)
	reqctx.SetVFSAccessScope(c, access.Scope)

	return c.Next()
}
