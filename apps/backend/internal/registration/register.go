package registration

import (
	"context"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/uptrace/bun"
)

// register uses transaction-bound stores for every provisioning step.
func (s *service) register(ctx context.Context, tx bun.Tx, opts user.CreateOptions) (*RegisterResult, error) {
	users := user.NewDBStore(tx)
	organizations := organization.NewDBStore(tx)
	accounts := account.NewDBStore(tx)
	drives := drive.NewDBStore(tx)

	u, err := users.Create(ctx, opts)
	if err != nil {
		return nil, err
	}

	org, err := organizations.CreatePersonal(ctx, "Personal")
	if err != nil {
		return nil, err
	}

	acc, err := accounts.Create(ctx, org.ID, u.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		return nil, err
	}

	drv, err := drives.Create(ctx, drive.CreateDriveOptions{
		OrgID:          org.ID,
		OwnerAccountID: &acc.ID,
		Name:           "My Drive",
		QuotaBytes:     1024 * 1024 * 1024, // 1GB
	})
	if err != nil {
		return nil, err
	}

	_, err = s.vfs.CreateRootDirectory(ctx, tx, drv.ID)
	if err != nil {
		return nil, err
	}

	tokens, err := s.authService.GrantForUser(ctx, tx, u)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		Account: acc,
		User:    u,
		Drive:   drv,
		Tokens:  tokens,
	}, nil
}
