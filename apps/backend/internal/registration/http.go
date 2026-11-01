package registration

import (
	"errors"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
)

type HTTPHandler struct {
	service      Service
	cookieConfig auth.CookieConfig
}

type registerAccountRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"displayName"`
	TokenDelivery string `json:"tokenDelivery" enums:"cookie,body"`
}

type registerAccountResponse struct {
	Account *account.Account `json:"account"`
	User    *user.User       `json:"user"`
	Drive   *drive.Drive     `json:"drive"`

	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

func NewHTTPHandler(service Service, cookieConfig auth.CookieConfig) *HTTPHandler {
	return &HTTPHandler{
		service:      service,
		cookieConfig: cookieConfig,
	}
}

func (h *HTTPHandler) RegisterRoutes(api fiber.Router) {
	api.Post("/accounts", h.registerAccount)
}

func (h *HTTPHandler) registerAccount(c *fiber.Ctx) error {
	req := new(registerAccountRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.TokenDelivery != auth.TokenDeliveryCookie && req.TokenDelivery != auth.TokenDeliveryBody {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid token delivery method"})
	}

	result, err := h.service.Register(c.Context(), RegisterOptions{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		var ae *user.AlreadyExistsError

		if errors.As(err, &ae) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "account already exists"})
		}

		if errors.Is(err, account.ErrAccountAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "account already exists"})
		}

		return httperr.Internal(err)
	}

	resp := registerAccountResponse{
		Account: result.Account,
		User:    result.User,
		Drive:   result.Drive,
	}

	if req.TokenDelivery == auth.TokenDeliveryCookie {
		auth.SetAuthCookies(c, result.Tokens.AccessToken, result.Tokens.RefreshToken, h.cookieConfig)
	} else {
		resp.AccessToken = result.Tokens.AccessToken
		resp.RefreshToken = result.Tokens.RefreshToken
	}

	return c.JSON(resp)
}
