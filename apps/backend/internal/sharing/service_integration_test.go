//go:build integration

package sharing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestService_SharingScopes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pg, err := runPostgres(ctx)
	if err != nil {
		t.Skipf("postgres testcontainer unavailable (docker not running/configured?): %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	postgresURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	db := database.NewFromPostgres(postgresURL)
	t.Cleanup(func() { _ = db.Close() })

	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	blobRoot, err := os.MkdirTemp("", "drexa-blobs-*")
	if err != nil {
		t.Fatalf("temp blob dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(blobRoot) })

	blobStore := blob.NewFSStore(blob.FSStoreConfig{Root: blobRoot})
	if err := blobStore.Initialize(ctx); err != nil {
		t.Fatalf("blob store init: %v", err)
	}

	vfs, err := virtualfs.New(blobStore, virtualfs.NewFlatKeyResolver())
	if err != nil {
		t.Fatalf("virtualfs.New: %v", err)
	}

	userSvc := user.NewService()
	orgSvc := organization.NewService()
	accSvc := account.NewService()
	driveSvc := drive.NewService()

	hashed, err := password.HashString("share-pass")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	testUser, err := userSvc.RegisterUser(ctx, db, user.UserRegistrationOptions{
		Email:       "share@example.com",
		DisplayName: "Share User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	org, err := orgSvc.CreatePersonalOrganization(ctx, db, "Personal")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}

	acc, err := accSvc.CreateAccount(ctx, db, org.ID, testUser.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	drv, err := driveSvc.CreateDrive(ctx, db, drive.CreateDriveOptions{
		OrgID:          org.ID,
		OwnerAccountID: &acc.ID,
		Name:           "My Drive",
		QuotaBytes:     1024 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("CreateDrive: %v", err)
	}

	root, err := vfs.CreateRootDirectory(ctx, db, drv.ID)
	if err != nil {
		t.Fatalf("CreateRootDirectory: %v", err)
	}

	scope := &virtualfs.Scope{
		DriveID:      drv.ID,
		RootNodeID:   root.ID,
		AllowedOps:   virtualfs.AllAllowedOps,
		AllowedNodes: nil,
		ActorKind:    virtualfs.ScopeActorAccount,
		ActorID:      acc.ID,
	}

	dirNode, err := vfs.CreateDirectory(ctx, db, root.ID, "Shared Folder", scope)
	if err != nil {
		t.Fatalf("CreateDirectory: %v", err)
	}

	fileNode, err := vfs.CreateFile(ctx, db, virtualfs.CreateFileOptions{
		ParentID: dirNode.ID,
		Name:     "shared.txt",
	}, scope)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if err := vfs.WriteFile(ctx, db, fileNode, virtualfs.FileContentFromReader(strings.NewReader("hello")), scope); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	shareSvc, err := NewService(vfs)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	t.Run("create share validation", func(t *testing.T) {
		if _, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{}); err != ErrShareNoItems {
			t.Fatalf("expected ErrShareNoItems, got %v", err)
		}

		if _, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items: []*virtualfs.Node{root},
		}); err != ErrCannotShareRoot {
			t.Fatalf("expected ErrCannotShareRoot, got %v", err)
		}

		if _, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items: []*virtualfs.Node{dirNode, fileNode},
		}); err != ErrNotSameParent {
			t.Fatalf("expected ErrNotSameParent, got %v", err)
		}
	})

	dirShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
		Items: []*virtualfs.Node{dirNode},
	})
	if err != nil {
		t.Fatalf("CreateShare(dir): %v", err)
	}
	fileShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
		Items: []*virtualfs.Node{fileNode},
	})
	if err != nil {
		t.Fatalf("CreateShare(file): %v", err)
	}

	t.Run("public scope for directory share", func(t *testing.T) {
		scope, err := shareSvc.ResolveScopeForShare(ctx, db, nil, dirShare)
		if err != nil {
			t.Fatalf("ResolveScopeForShare(dir): %v", err)
		}
		if scope.ActorKind != virtualfs.ScopeActorShare {
			t.Fatalf("unexpected actor kind: got %q want %q", scope.ActorKind, virtualfs.ScopeActorShare)
		}
		if scope.ActorID != dirShare.ID {
			t.Fatalf("unexpected actor id: got %q want %q", scope.ActorID, dirShare.ID)
		}
		if scope.RootNodeID != dirShare.SharedDirectoryID {
			t.Fatalf("unexpected root node: got %q want %q", scope.RootNodeID, dirShare.SharedDirectoryID)
		}
		if !scope.Allows(virtualfs.OperationRead) {
			t.Fatalf("expected read to be allowed")
		}
		if scope.Allows(virtualfs.OperationWrite) || scope.Allows(virtualfs.OperationUpload) {
			t.Fatalf("expected write/upload to be disallowed")
		}
		if _, ok := scope.AllowedNodes[dirNode.ID]; !ok {
			t.Fatalf("expected directory node to be allowlisted")
		}
	})

	t.Run("public scope for file share", func(t *testing.T) {
		scope, err := shareSvc.ResolveScopeForShare(ctx, db, nil, fileShare)
		if err != nil {
			t.Fatalf("ResolveScopeForShare(file): %v", err)
		}
		if scope.ActorKind != virtualfs.ScopeActorShare {
			t.Fatalf("unexpected actor kind: got %q want %q", scope.ActorKind, virtualfs.ScopeActorShare)
		}
		if scope.ActorID != fileShare.ID {
			t.Fatalf("unexpected actor id: got %q want %q", scope.ActorID, fileShare.ID)
		}
		if scope.RootNodeID != dirNode.ID {
			t.Fatalf("unexpected root node: got %q want %q", scope.RootNodeID, dirNode.ID)
		}
		if scope.RootNodeID != fileShare.SharedDirectoryID {
			t.Fatalf("unexpected root node: got %q want %q", scope.RootNodeID, fileShare.SharedDirectoryID)
		}
		if _, ok := scope.AllowedNodes[fileNode.ID]; !ok {
			t.Fatalf("expected file node to be allowlisted")
		}
	})

	t.Run("list shares includes expired", func(t *testing.T) {
		expiredShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items:     []*virtualfs.Node{fileNode},
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateShare(expired list): %v", err)
		}

		activeShares, err := shareSvc.ListShares(ctx, db, drv.ID, ListSharesOptions{IncludesExpired: false})
		if err != nil {
			t.Fatalf("ListShares(active): %v", err)
		}
		activeIDs := make(map[string]bool, len(activeShares))
		for _, sh := range activeShares {
			activeIDs[sh.ID.String()] = true
		}
		if activeIDs[expiredShare.ID.String()] {
			t.Fatalf("expected expired share to be excluded")
		}
		if !activeIDs[dirShare.ID.String()] || !activeIDs[fileShare.ID.String()] {
			t.Fatalf("expected active shares to be listed")
		}

		allShares, err := shareSvc.ListShares(ctx, db, drv.ID, ListSharesOptions{IncludesExpired: true})
		if err != nil {
			t.Fatalf("ListShares(all): %v", err)
		}
		allIDs := make(map[string]bool, len(allShares))
		for _, sh := range allShares {
			allIDs[sh.ID.String()] = true
		}
		if !allIDs[expiredShare.ID.String()] {
			t.Fatalf("expected expired share to be included")
		}
		if !allIDs[dirShare.ID.String()] || !allIDs[fileShare.ID.String()] {
			t.Fatalf("expected active shares to be listed")
		}
	})

	t.Run("find share not found", func(t *testing.T) {
		if _, err := shareSvc.FindShareByPublicID(ctx, db, "missing-share"); err != ErrShareNotFound {
			t.Fatalf("expected ErrShareNotFound, got %v", err)
		}
	})

	t.Run("expired share", func(t *testing.T) {
		expiredShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items:     []*virtualfs.Node{fileNode},
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateShare(expired): %v", err)
		}
		if _, err := shareSvc.ResolveScopeForShare(ctx, db, nil, expiredShare); err != ErrShareExpired {
			t.Fatalf("expected ErrShareExpired, got %v", err)
		}
	})

	t.Run("revoked share", func(t *testing.T) {
		revokedShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(revoked): %v", err)
		}
		now := time.Now()
		if _, err := db.NewUpdate().Model(revokedShare).
			Set("revoked_at = ?", now).
			WherePK().
			Exec(ctx); err != nil {
			t.Fatalf("update revoked share: %v", err)
		}
		revokedShare.RevokedAt = &now
		if _, err := shareSvc.ResolveScopeForShare(ctx, db, nil, revokedShare); err != ErrShareRevoked {
			t.Fatalf("expected ErrShareRevoked, got %v", err)
		}
	})

	t.Run("no permissions", func(t *testing.T) {
		noPermShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(no permissions): %v", err)
		}
		if _, err := db.NewDelete().
			Model(&SharePermission{}).
			Where("share_id = ?", noPermShare.ID).
			Exec(ctx); err != nil {
			t.Fatalf("delete permissions: %v", err)
		}
		if _, err := shareSvc.ResolveScopeForShare(ctx, db, nil, noPermShare); err != ErrNoPermissions {
			t.Fatalf("expected ErrNoPermissions, got %v", err)
		}
	})

	t.Run("permission expired", func(t *testing.T) {
		expiredPermShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(expired perm): %v", err)
		}
		expiredAt := time.Now().Add(-1 * time.Hour)
		if _, err := db.NewUpdate().
			Model(&SharePermission{}).
			Set("expires_at = ?", expiredAt).
			Where("share_id = ?", expiredPermShare.ID).
			Exec(ctx); err != nil {
			t.Fatalf("update permission expiry: %v", err)
		}
		if _, err := shareSvc.ResolveScopeForShare(ctx, db, nil, expiredPermShare); err != ErrShareExpired {
			t.Fatalf("expected ErrShareExpired, got %v", err)
		}
	})

	t.Run("no items", func(t *testing.T) {
		noItemsShare, err := shareSvc.CreateShare(ctx, db, drv.ID, acc.ID, CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(no items): %v", err)
		}
		if _, err := db.NewDelete().
			Model(&ShareItem{}).
			Where("share_id = ?", noItemsShare.ID).
			Exec(ctx); err != nil {
			t.Fatalf("delete share items: %v", err)
		}
		if _, err := shareSvc.ResolveScopeForShare(ctx, db, nil, noItemsShare); err != ErrShareNoItems {
			t.Fatalf("expected ErrShareNoItems, got %v", err)
		}
	})
}

func runPostgres(ctx context.Context) (_ *postgres.PostgresContainer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("testcontainers panic: %v", r)
		}
	}()

	return postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("drexa"),
		postgres.WithUsername("drexa"),
		postgres.WithPassword("drexa"),
		postgres.BasicWaitStrategies(),
	)
}
