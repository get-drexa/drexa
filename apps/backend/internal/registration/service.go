package registration

import (
	"context"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/uptrace/bun"
)

type Service struct {
	userService         user.Service
	organizationService organization.Service
	accountService      account.Service
	driveService        drive.Service
	vfs                *virtualfs.VirtualFS
}

type RegisterOptions struct {
	Email       string
	Password    string
	DisplayName string
}

type RegisterResult struct {
	Account   *account.Account
	User      *user.User
	Drive     *drive.Drive
}

func NewService(userService *user.Service, organizationService *organization.Service, accountService *account.Service, driveService *drive.Service, vfs *virtualfs.VirtualFS) *Service {
	return &Service{
		userService:         *userService,
		organizationService: *organizationService,
		accountService:      *accountService,
		driveService:        *driveService,
		vfs:                vfs,
	}
}

func (s *Service) Register(ctx context.Context, db bun.IDB, opts RegisterOptions) (*RegisterResult, error) {
	hashed, err := password.HashString(opts.Password)
	if err != nil {
		return nil, err
	}

	u, err := s.userService.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       opts.Email,
		Password:    hashed,
		DisplayName: opts.DisplayName,
	})
	if err != nil {
		return nil, err
	}

	org, err := s.organizationService.CreatePersonalOrganization(ctx, db, "Personal")
	if err != nil {
		return nil, err
	}

	acc, err := s.accountService.CreateAccount(ctx, db, org.ID, u.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		return nil, err
	}

	drv, err := s.driveService.CreateDrive(ctx, db, drive.CreateDriveOptions{
		OrgID:          org.ID,
		OwnerAccountID: &acc.ID,
		Name:           "My Drive",
		QuotaBytes:     1024 * 1024 * 1024, // 1GB
	})
	if err != nil {
		return nil, err
	}

	_, err = s.vfs.CreateRootDirectory(ctx, db, drv.ID)
	if err != nil {
		return nil, err
	}

	return &RegisterResult{
		Account:   acc,
		User:      u,
		Drive:     drv,
	}, nil
}
