package account

import (
	"errors"

	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	accountService *Service
	authService    *auth.Service
	vfs            *virtualfs.VirtualFS
	db             *bun.DB
	authMiddleware fiber.Handler
	cookieConfig   auth.CookieConfig
}

// registerAccountRequest represents a new account registration
// @Description Request to create a new account and user
type registerAccountRequest struct {
	// Email address for the new account
	Email string `json:"email" example:"newuser@example.com"`
	// Password for the new account (min 8 characters)
	Password string `json:"password" example:"securepassword123"`
	// Display name for the user
	DisplayName string `json:"displayName" example:"Jane Doe"`
	// How to deliver tokens: "cookie" (set HTTP-only cookies) or "body" (include in response)
	TokenDelivery string `json:"tokenDelivery" example:"body" enums:"cookie,body"`
}

// registerAccountResponse represents a successful registration
// @Description Response after successful account registration
type registerAccountResponse struct {
	// The created account
	Account *Account `json:"account"`
	// The created user
	User *user.User `json:"user"`
	// JWT access token for immediate authentication (only included when tokenDelivery is "body")
	AccessToken string `json:"accessToken,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.signature"`
	// Base64 URL encoded refresh token (only included when tokenDelivery is "body")
	RefreshToken string `json:"refreshToken,omitempty" example:"dR4nD0mUu1DkZXlCeXRlc0FuZFJhbmRvbURhdGFIZXJlMTIzNDU2Nzg5MGFi"`
}

func NewHTTPHandler(accountService *Service, authService *auth.Service, vfs *virtualfs.VirtualFS, db *bun.DB, authMiddleware fiber.Handler, cookieConfig auth.CookieConfig) *HTTPHandler {
	return &HTTPHandler{accountService: accountService, authService: authService, db: db, authMiddleware: authMiddleware, cookieConfig: cookieConfig}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) *ScopedRouter {
	api.Get("/accounts", h.authMiddleware, h.listAccounts)
	api.Post("/accounts", h.registerAccount)

	account := api.Group("/accounts/:accountID")
	account.Use(h.authMiddleware)
	account.Use(h.accountMiddleware)

	account.Get("/", h.getAccount)

	return &ScopedRouter{virtualfs.ScopedRouter{account}}
}

func (h *HTTPHandler) accountMiddleware(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*user.User)

	accountID, err := uuid.Parse(c.Params("accountID"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	account, err := h.accountService.AccountByID(c.Context(), h.db, u.ID, accountID)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return httperr.Internal(err)
	}

	root, err := h.vfs.FindRootDirectory(c.Context(), h.db, account.ID)
	if err != nil {
		return httperr.Internal(err)
	}

	scope := &virtualfs.Scope{
		AccountID:    account.ID,
		RootNodeID:   root.ID,
		AllowedOps:   virtualfs.AllAllowedOps,
		AllowedNodes: nil,
		ActorKind:    virtualfs.ScopeActorAccount,
		ActorID:      u.ID,
	}

	reqctx.SetVFSAccessScope(c, scope)
	reqctx.SetCurrentAccount(c, account)

	return c.Next()
}

// listAccounts lists all accounts for the authenticated user
// @Summary List accounts
// @Description Retrieve all accounts for the authenticated user
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Success 200 {array} Account "List of accounts for the authenticated user"
// @Failure 401 {string} string "Not authenticated"
// @Router /accounts [get]
func (h *HTTPHandler) listAccounts(c *fiber.Ctx) error {
	u := reqctx.AuthenticatedUser(c).(*user.User)
	accounts, err := h.accountService.ListAccounts(c.Context(), h.db, u.ID)
	if err != nil {
		return httperr.Internal(err)
	}
	return c.JSON(accounts)
}

// getAccount retrieves account information
// @Summary Get account
// @Description Retrieve account details including storage usage and quota
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID" format(uuid)
// @Success 200 {object} Account "Account details"
// @Failure 401 {string} string "Not authenticated"
// @Failure 404 {string} string "Account not found"
// @Router /accounts/{accountID} [get]
func (h *HTTPHandler) getAccount(c *fiber.Ctx) error {
	account, ok := reqctx.CurrentAccount(c).(*Account)
	if !ok || account == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.JSON(account)
}

// registerAccount creates a new account and user
// @Summary Register new account
// @Description Create a new user account with email and password. Returns the account, user, and authentication tokens. Tokens can be delivered via HTTP-only cookies or in the response body based on the tokenDelivery field.
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body registerAccountRequest true "Registration details"
// @Success 200 {object} registerAccountResponse "Account created successfully"
// @Failure 400 {string} string "Invalid request body or token delivery method"
// @Failure 409 {string} string "Email already registered"
// @Router /accounts [post]
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

	result, err := h.authService.GrantForUser(c.Context(), tx, u)
	if err != nil {
		return httperr.Internal(err)
	}

	err = tx.Commit()
	if err != nil {
		return httperr.Internal(err)
	}

	switch req.TokenDelivery {
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid token delivery method"})

	case auth.TokenDeliveryCookie:
		auth.SetAuthCookies(c, result.AccessToken, result.RefreshToken, h.cookieConfig)
		return c.JSON(registerAccountResponse{
			Account: acc,
			User:    u,
		})

	case auth.TokenDeliveryBody:
		return c.JSON(registerAccountResponse{
			Account:      acc,
			User:         u,
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
		})
	}
}
