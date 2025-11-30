package account

import (
	"errors"

	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	accountService *Service
	authService    *auth.Service
	db             *bun.DB
	authMiddleware fiber.Handler
}

type registerAccountRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type registerAccountResponse struct {
	Account      *Account   `json:"account"`
	User         *user.User `json:"user"`
	AccessToken  string     `json:"accessToken"`
	RefreshToken string     `json:"refreshToken"`
}

const currentAccountKey = "currentAccount"

func CurrentAccount(c *fiber.Ctx) *Account {
	return c.Locals(currentAccountKey).(*Account)
}

func NewHTTPHandler(accountService *Service, authService *auth.Service, db *bun.DB, authMiddleware fiber.Handler) *HTTPHandler {
	return &HTTPHandler{accountService: accountService, authService: authService, db: db, authMiddleware: authMiddleware}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) fiber.Router {
	api.Post("/accounts", h.registerAccount)

	account := api.Group("/accounts/:accountID")
	account.Use(h.authMiddleware)
	account.Use(h.accountMiddleware)

	account.Get("/", h.getAccount)

	return account
}

func (h *HTTPHandler) accountMiddleware(c *fiber.Ctx) error {
	user, err := auth.AuthenticatedUser(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	accountID, err := uuid.Parse(c.Params("accountID"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	account, err := h.accountService.AccountByID(c.Context(), h.db, user.ID, accountID)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	c.Locals(currentAccountKey, account)

	return c.Next()
}

func (h *HTTPHandler) getAccount(c *fiber.Ctx) error {
	account := CurrentAccount(c)
	if account == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.JSON(account)
}

func (h *HTTPHandler) registerAccount(c *fiber.Ctx) error {
	req := new(registerAccountRequest)
	if err := c.BodyParser(req); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return httperr.Internal(err)
	}
	defer tx.Rollback()

	acc, u, err := h.accountService.Register(c.Context(), tx, RegisterOptions{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		var ae *user.AlreadyExistsError
		if errors.As(err, &ae) {
			return c.SendStatus(fiber.StatusConflict)
		}
		if errors.Is(err, ErrAccountAlreadyExists) {
			return c.SendStatus(fiber.StatusConflict)
		}
		return httperr.Internal(err)
	}

	result, err := h.authService.GenerateTokenForUser(c.Context(), tx, u)
	if err != nil {
		return httperr.Internal(err)
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	return c.JSON(registerAccountResponse{
		Account:      acc,
		User:         u,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
