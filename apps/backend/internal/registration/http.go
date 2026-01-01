package registration

import (
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

type HTTPHandler struct {
	service      *Service
	authService   *auth.Service
	db            *bun.DB
	cookieConfig  auth.CookieConfig
}

type registerAccountRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"displayName"`
	TokenDelivery string `json:"tokenDelivery" enums:"cookie,body"`
}

type registerAccountResponse struct {
	Account   *account.Account     `json:"account"`
	User      *user.User           `json:"user"`
	Drive     *drive.Drive         `json:"drive"`

	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

func NewHTTPHandler(service *Service, authService *auth.Service, db *bun.DB, cookieConfig auth.CookieConfig) *HTTPHandler {
	return &HTTPHandler{
		service:     service,
		authService: authService,
		db:          db,
		cookieConfig: cookieConfig,
	}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	api.Post("/accounts", h.registerAccount)
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

	result, err := h.service.Register(c.Context(), tx, RegisterOptions{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		var ae *user.AlreadyExistsError
		if errors.As(err, &ae) {
			return c.SendStatus(fiber.StatusConflict)
		}
		if errors.Is(err, account.ErrAccountAlreadyExists) {
			return c.SendStatus(fiber.StatusConflict)
		}
		return httperr.Internal(err)
	}

	grant, err := h.authService.GrantForUser(c.Context(), tx, result.User)
	if err != nil {
		return httperr.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return httperr.Internal(err)
	}

	resp := registerAccountResponse{
		Account:   result.Account,
		User:      result.User,
		Drive:     result.Drive,
	}

	switch req.TokenDelivery {
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid token delivery method"})

	case auth.TokenDeliveryCookie:
		auth.SetAuthCookies(c, grant.AccessToken, grant.RefreshToken, h.cookieConfig)
		return c.JSON(resp)

	case auth.TokenDeliveryBody:
		resp.AccessToken = grant.AccessToken
		resp.RefreshToken = grant.RefreshToken
		return c.JSON(resp)
	}
}
