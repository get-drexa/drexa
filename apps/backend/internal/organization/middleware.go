package organization

import (
	"errors"
	"strings"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/reqctx"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

func NewMiddleware(orgService *Service, accountService *account.Service, db *bun.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := strings.ToLower(c.Params("orgSlug"))
		if slug == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		u, _ := reqctx.AuthenticatedUser(c).(*user.User)
		if u == nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		var org *Organization
		var err error

		if slug == ReservedSlug {
			org, err = orgService.PersonalOrganizationForUser(c.Context(), db, u.ID)
		} else {
			org, err = orgService.OrganizationBySlug(c.Context(), db, slug)
		}

		if err != nil {
			if errors.Is(err, ErrOrganizationNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}

		acc, err := accountService.FindUserAccountInOrg(c.Context(), db, org.ID, u.ID)
		if err != nil {
			if errors.Is(err, account.ErrAccountNotFound) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return httperr.Internal(err)
		}
		if acc.Status != account.StatusActive {
			return c.SendStatus(fiber.StatusNotFound)
		}
		reqctx.SetCurrentAccount(c, acc)

		reqctx.SetCurrentOrganization(c, org)

		return c.Next()
	}
}
