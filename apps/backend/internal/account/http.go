package account

import (
	"errors"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	accountService  *Service
	db              *bun.DB
	authMiddleware  fiber.Handler
}

func NewHTTPHandler(accountService *Service, db *bun.DB, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{
		accountService:  accountService,
		db:              db,
		authMiddleware:  authMiddleware,
	}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	api.Get("/accounts", h.authMiddleware, h.listAccounts)
	api.Get("/accounts/:accountID", h.authMiddleware, h.getAccount)
}

func (h *HTTPHandler) listAccounts(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*user.User)
	accounts, err := h.accountService.ListAccounts(c.Context(), h.db, u.ID)
	if err != nil {
		return httperr.Internal(err)
	}
	return c.JSON(accounts)
}

func (h *HTTPHandler) getAccount(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*user.User)
	accountID, err := uuid.Parse(c.Params("accountID"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	acc, err := h.accountService.AccountByID(c.Context(), h.db, u.ID, accountID)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}
	return c.JSON(acc)
}
