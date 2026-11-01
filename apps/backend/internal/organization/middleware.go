package organization

import (
	"errors"

	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
)

func NewMiddleware(service Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		u, _ := reqctx.AuthenticatedUser(c).(*user.User)
		if u == nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		membership, err := service.ResolveForUser(c.Context(), c.Params("orgSlug"), u.ID)
		if err != nil {
			if errors.Is(err, ErrOrganizationNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}

			return httperr.Internal(err)
		}

		reqctx.SetCurrentAccount(c, membership.Account)
		reqctx.SetCurrentOrganization(c, membership.Organization)

		return c.Next()
	}
}
