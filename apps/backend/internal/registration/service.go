package registration

import (
	"context"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/uptrace/bun"
)

type Service interface {
	Register(ctx context.Context, opts RegisterOptions) (*RegisterResult, error)
}

type service struct {
	db          *bun.DB
	authService auth.Service
	vfs         *virtualfs.VirtualFS
}

var _ Service = (*service)(nil)

type RegisterOptions struct {
	Email       string
	Password    string
	DisplayName string
}

type RegisterResult struct {
	Account *account.Account
	User    *user.User
	Drive   *drive.Drive
	Tokens  *auth.AuthenticationTokens
}

func NewService(db *bun.DB, authService auth.Service, vfs *virtualfs.VirtualFS) Service {
	return &service{db: db, authService: authService, vfs: vfs}
}

// Register provisions a user and their personal workspace, then issues auth
// tokens. All database writes commit together before a result is returned.
func (s *service) Register(ctx context.Context, opts RegisterOptions) (*RegisterResult, error) {
	hashed, err := password.HashString(opts.Password)
	if err != nil {
		return nil, err
	}

	var result *RegisterResult

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var err error

		result, err = s.register(ctx, tx, user.CreateOptions{
			Email:       opts.Email,
			Password:    hashed,
			DisplayName: opts.DisplayName,
		})

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
