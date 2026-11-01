//go:build integration

package sharing_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/password"
	"github.com/get-drexa/drexa/internal/sharing"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/google/uuid"
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

	userSvc := user.NewService(user.NewDBStore(db))
	orgSvc := organization.NewService(organization.NewDBStore(db), account.NewDBStore(db))
	accSvc := account.NewService(account.NewDBStore(db))
	driveSvc := drive.NewService(drive.NewDBStore(db), account.NewDBStore(db), db, vfs)

	hashed, err := password.HashString("share-pass")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}

	testUser, err := userSvc.RegisterUser(ctx, user.UserRegistrationOptions{
		Email:       "share@example.com",
		DisplayName: "Share User",
		Password:    hashed,
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	org, err := orgSvc.CreatePersonalOrganization(ctx, "Personal")
	if err != nil {
		t.Fatalf("CreatePersonalOrganization: %v", err)
	}

	acc, err := accSvc.CreateAccount(ctx, org.ID, testUser.ID, account.RoleAdmin, account.StatusActive)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	drv, err := driveSvc.CreateDrive(ctx, drive.CreateDriveOptions{
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

	shareSvc, err := sharing.NewService(db, vfs)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	t.Run("create share validation", func(t *testing.T) {
		if _, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{}); err != sharing.ErrShareNoItems {
			t.Fatalf("expected ErrShareNoItems, got %v", err)
		}

		if _, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items: []*virtualfs.Node{root},
		}); err != sharing.ErrCannotShareRoot {
			t.Fatalf("expected ErrCannotShareRoot, got %v", err)
		}

		if _, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items: []*virtualfs.Node{dirNode, fileNode},
		}); err != sharing.ErrNotSameParent {
			t.Fatalf("expected ErrNotSameParent, got %v", err)
		}
	})

	dirShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
		Items: []*virtualfs.Node{dirNode},
	})
	if err != nil {
		t.Fatalf("CreateShare(dir): %v", err)
	}

	fileShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
		Items: []*virtualfs.Node{fileNode},
	})
	if err != nil {
		t.Fatalf("CreateShare(file): %v", err)
	}

	t.Run("public scope for directory share", func(t *testing.T) {
		scope, err := shareSvc.ResolveScopeForShare(ctx, nil, dirShare)
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
		scope, err := shareSvc.ResolveScopeForShare(ctx, nil, fileShare)
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
		expiredShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items:     []*virtualfs.Node{fileNode},
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateShare(expired list): %v", err)
		}

		activeShares, err := shareSvc.ListShares(ctx, drv.ID, sharing.ListSharesOptions{IncludesExpired: false})
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

		allShares, err := shareSvc.ListShares(ctx, drv.ID, sharing.ListSharesOptions{IncludesExpired: true})
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
		if _, err := shareSvc.FindShareByPublicID(ctx, "missing-share"); err != sharing.ErrShareNotFound {
			t.Fatalf("expected ErrShareNotFound, got %v", err)
		}
	})

	t.Run("expired share", func(t *testing.T) {
		expiredShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items:     []*virtualfs.Node{fileNode},
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateShare(expired): %v", err)
		}

		if _, err := shareSvc.ResolveScopeForShare(ctx, nil, expiredShare); err != sharing.ErrShareExpired {
			t.Fatalf("expected ErrShareExpired, got %v", err)
		}
	})

	t.Run("revoked share", func(t *testing.T) {
		revokedShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
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
		if _, err := shareSvc.ResolveScopeForShare(ctx, nil, revokedShare); err != sharing.ErrShareRevoked {
			t.Fatalf("expected ErrShareRevoked, got %v", err)
		}
	})

	t.Run("no permissions", func(t *testing.T) {
		noPermShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(no permissions): %v", err)
		}

		if _, err := db.NewDelete().
			Model(&sharing.SharePermission{}).
			Where("share_id = ?", noPermShare.ID).
			Exec(ctx); err != nil {
			t.Fatalf("delete permissions: %v", err)
		}

		if _, err := shareSvc.ResolveScopeForShare(ctx, nil, noPermShare); err != sharing.ErrNoPermissions {
			t.Fatalf("expected ErrNoPermissions, got %v", err)
		}
	})

	t.Run("permission expired", func(t *testing.T) {
		expiredPermShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(expired perm): %v", err)
		}

		expiredAt := time.Now().Add(-1 * time.Hour)
		if _, err := db.NewUpdate().
			Model(&sharing.SharePermission{}).
			Set("expires_at = ?", expiredAt).
			Where("share_id = ?", expiredPermShare.ID).
			Exec(ctx); err != nil {
			t.Fatalf("update permission expiry: %v", err)
		}

		if _, err := shareSvc.ResolveScopeForShare(ctx, nil, expiredPermShare); err != sharing.ErrShareExpired {
			t.Fatalf("expected ErrShareExpired, got %v", err)
		}
	})

	t.Run("no items", func(t *testing.T) {
		noItemsShare, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{
			Items: []*virtualfs.Node{fileNode},
		})
		if err != nil {
			t.Fatalf("CreateShare(no items): %v", err)
		}

		if _, err := db.NewDelete().
			Model(&sharing.ShareItem{}).
			Where("share_id = ?", noItemsShare.ID).
			Exec(ctx); err != nil {
			t.Fatalf("delete share items: %v", err)
		}

		if _, err := shareSvc.ResolveScopeForShare(ctx, nil, noItemsShare); err != sharing.ErrShareNoItems {
			t.Fatalf("expected ErrShareNoItems, got %v", err)
		}
	})
	t.Run("account-only share requires the matching active account", func(t *testing.T) {
		sh, err := shareSvc.CreateShare(ctx, drv.ID, acc.ID, sharing.CreateShareOptions{Items: []*virtualfs.Node{fileNode}})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := db.NewUpdate().Model((*sharing.SharePermission)(nil)).Set("account_id = ?", acc.ID).Where("share_id = ?", sh.ID).Exec(ctx); err != nil {
			t.Fatal(err)
		}

		other := *acc
		other.ID = uuid.New()
		suspended := *acc
		suspended.Status = account.StatusSuspended
		for _, consumer := range []*account.Account{nil, &other, &suspended} {
			if got, err := shareSvc.ResolveScopeForShare(ctx, consumer, sh); !errors.Is(err, sharing.ErrNoPermissions) || got != nil {
				t.Fatalf("unauthorized consumer received scope: %v", err)
			}
		}

		if got, err := shareSvc.ResolveAccess(ctx, sh.PublicID, testUser.ID, org.ID); err != nil || got.ActorID != acc.ID {
			t.Fatalf("matching consumer could not resolve share: %v", err)
		}

		if _, err := shareSvc.ResolveAccess(ctx, sh.PublicID, uuid.Nil, uuid.Nil); !errors.Is(err, sharing.ErrShareNotFound) {
			t.Fatalf("anonymous access was not hidden: %v", err)
		}
	})

	t.Run("create share rolls back all records on permission failure", func(t *testing.T) {
		counts := func() []int {
			var result []int

			for _, table := range []string{"node_shares", "share_items", "share_permissions"} {
				count, err := db.NewSelect().Table(table).Count(ctx)
				if err != nil {
					t.Fatal(err)
				}

				result = append(result, count)
			}

			return result
		}

		before := counts()
		if _, err := db.ExecContext(ctx, "ALTER TABLE share_permissions ADD CONSTRAINT reject_permission CHECK (false) NOT VALID"); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _, _ = db.ExecContext(ctx, "ALTER TABLE share_permissions DROP CONSTRAINT reject_permission") })
		sh, err := shareSvc.CreateShareForItems(ctx, drv.ID, acc, []string{fileNode.PublicID}, time.Time{}, scope)
		if err == nil || sh != nil {
			t.Fatal("expected share creation failure without a result")
		}

		if after := counts(); !reflect.DeepEqual(before, after) {
			t.Fatalf("partial share persisted: before=%v after=%v", before, after)
		}
	})

	t.Run("share management checks scope and ownership", func(t *testing.T) {
		sh, err := shareSvc.CreateShareForItems(ctx, drv.ID, acc, []string{fileNode.PublicID}, time.Time{}, scope)
		if err != nil {
			t.Fatal(err)
		}

		other := *acc
		other.ID = uuid.New()
		other.Role = account.RoleMember
		expires := time.Now().Add(time.Hour)
		if _, err := shareSvc.UpdateShareForAccount(ctx, drv.ID, &other, sh.PublicID, sharing.UpdateShareOptions{ExpiresAt: &expires}); !errors.Is(err, sharing.ErrShareNotFound) {
			t.Fatalf("non-owner updated share: %v", err)
		}

		if err := shareSvc.DeleteShareForAccount(ctx, uuid.New(), acc, sh.PublicID); !errors.Is(err, sharing.ErrShareNotFound) {
			t.Fatalf("wrong drive deleted share: %v", err)
		}

		updated, err := shareSvc.UpdateShareForAccount(ctx, drv.ID, acc, sh.PublicID, sharing.UpdateShareOptions{ExpiresAt: &expires})
		if err != nil || updated.ExpiresAt == nil {
			t.Fatalf("owner could not update share: %v", err)
		}

		updated, err = shareSvc.UpdateShareForAccount(ctx, drv.ID, acc, sh.PublicID, sharing.UpdateShareOptions{ExpiresAt: &time.Time{}})
		if err != nil || updated.ExpiresAt != nil {
			t.Fatalf("owner could not clear expiry: %v", err)
		}

		if err := shareSvc.DeleteShareForAccount(ctx, drv.ID, acc, sh.PublicID); err != nil {
			t.Fatal(err)
		}

		if _, err := shareSvc.FindShareByPublicID(ctx, sh.PublicID); !errors.Is(err, sharing.ErrShareNotFound) {
			t.Fatalf("share survived deletion: %v", err)
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
