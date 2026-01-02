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
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	driveService   *Service
	accountService *account.Service
	vfs            *virtualfs.VirtualFS
	db             *bun.DB
	authMiddleware fiber.Handler
}

func NewHTTPHandler(driveService *Service, accountService *account.Service, vfs *virtualfs.VirtualFS, db *bun.DB, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{
		driveService:   driveService,
		accountService: accountService,
		vfs:            vfs,
		db:             db,
		authMiddleware: authMiddleware,
	}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) *virtualfs.ScopedRouter {
	api.Get("/drives", h.authMiddleware, h.listDrives)

	drive := api.Group("/drives/:driveID")
	drive.Use(h.authMiddleware)
	drive.Use(h.driveMiddleware)

	drive.Get("/", h.getDrive)

	return &virtualfs.ScopedRouter{Router: drive}
}

func (h *HTTPHandler) listDrives(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*user.User)
	org, ok := reqctx.CurrentOrganization(c).(*organization.Organization)
	if !ok || org == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	acc, err := h.accountService.FindUserAccountInOrg(c.Context(), h.db, org.ID, u.ID)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}
	if acc.Status != account.StatusActive {
		return c.SendStatus(fiber.StatusNotFound)
	}

	drives, err := h.driveService.ListAccessibleDrives(c.Context(), h.db, org.ID, acc.ID)
	if err != nil {
		return httperr.Internal(err)
	}
	return c.JSON(drives)
}

func (h *HTTPHandler) getDrive(c *fiber.Ctx) error {
	drive, ok := reqctx.CurrentDrive(c).(*Drive)
	if !ok || drive == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.JSON(drive)
}

func (h *HTTPHandler) driveMiddleware(c *fiber.Ctx) error {
	org, ok := reqctx.CurrentOrganization(c).(*organization.Organization)
	if !ok || org == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	driveID, err := uuid.Parse(c.Params("driveID"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	drive, err := h.driveService.DriveByID(c.Context(), h.db, driveID)
	if err != nil {
		if errors.Is(err, ErrDriveNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}
	if drive.OrgID != org.ID {
		return c.SendStatus(fiber.StatusNotFound)
	}

	acc, ok := reqctx.CurrentAccount(c).(*account.Account)
	if !ok || acc == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !h.driveService.CanAccessDrive(drive, acc.OrgID, acc.ID) {
		return c.SendStatus(fiber.StatusNotFound)
	}

	root, err := h.vfs.FindRootDirectory(c.Context(), h.db, drive.ID)
	if err != nil {
		return httperr.Internal(err)
	}

	scope := &virtualfs.Scope{
		DriveID:      drive.ID,
		RootNodeID:   root.ID,
		AllowedOps:   virtualfs.AllAllowedOps,
		AllowedNodes: nil,
		ActorKind:    virtualfs.ScopeActorAccount,
		ActorID:      acc.ID,
	}

	reqctx.SetCurrentDrive(c, drive)
	reqctx.SetVFSAccessScope(c, scope)

	return c.Next()
}
