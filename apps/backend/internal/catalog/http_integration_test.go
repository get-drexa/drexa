//go:build integration

package catalog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/catalog"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/drive"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/organization"
	"github.com/get-drexa/drexa/internal/registration"
	"github.com/get-drexa/drexa/internal/sharing"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"
)

type testEnv struct {
	db          *bun.DB
	vfs         *virtualfs.VirtualFS
	app         *fiber.App
	accessToken string
	driveID     string
	rootDirID   uuid.UUID
	scope       *virtualfs.Scope
	orgSlug     string
}

func setupTestEnv(t *testing.T, ctx context.Context) *testEnv {
	t.Helper()

	pg, err := runPostgres(ctx)
	if err != nil {
		t.Skipf("postgres testcontainer unavailable: %v", err)
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

	orgSvc := organization.NewService(organization.NewDBStore(db), account.NewDBStore(db))
	driveSvc := drive.NewService(drive.NewDBStore(db), account.NewDBStore(db), db, vfs)
	sharingSvc, err := sharing.NewService(db, vfs)
	if err != nil {
		t.Fatalf("sharing.NewService: %v", err)
	}

	authSvc := auth.NewService(db, user.NewDBStore(db), auth.TokenConfig{
		Issuer:    "drexa-test",
		Audience:  "drexa-test",
		SecretKey: []byte("drexa-test-secret"),
	})
	regSvc := registration.NewService(db, authSvc, vfs)

	regResult, err := regSvc.Register(ctx, registration.RegisterOptions{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	grant := regResult.Tokens

	authMiddleware := auth.NewAuthMiddleware(authSvc, auth.CookieConfig{})

	root, err := vfs.FindRootDirectory(ctx, db, regResult.Drive.ID)
	if err != nil {
		t.Fatalf("FindRootDirectory: %v", err)
	}

	scope := &virtualfs.Scope{
		DriveID:    regResult.Drive.ID,
		RootNodeID: root.ID,
		AllowedOps: virtualfs.AllAllowedOps,
		ActorKind:  virtualfs.ScopeActorAccount,
		ActorID:    regResult.Account.ID,
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: httperr.ErrorHandler,
	})
	api := app.Group("/api")

	// Set up org-scoped routes like the real server does
	orgMiddleware := organization.NewMiddleware(orgSvc)
	orgAPI := api.Group("/:orgSlug", authMiddleware, orgMiddleware)

	driveHTTP := drive.NewHTTPHandler(driveSvc, authMiddleware)
	scopedRouter := driveHTTP.RegisterRoutes(orgAPI)

	catalogHTTP := catalog.NewHTTPHandler(catalog.NewService(db, vfs), sharingSvc)
	catalogHTTP.RegisterRoutes(scopedRouter)

	return &testEnv{
		db:          db,
		vfs:         vfs,
		app:         app,
		accessToken: grant.AccessToken,
		driveID:     regResult.Drive.PublicID,
		rootDirID:   root.ID,
		scope:       scope,
		orgSlug:     organization.ReservedSlug,
	}
}

func TestHTTP_CatalogEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	env := setupTestEnv(t, ctx)

	// Create test data using VFS directly:
	// - 2 files in root: file1.txt, file2.txt
	// - 1 directory in root: subdir
	// - 1 file in subdir: nested.txt

	file1, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
		ParentID: env.rootDirID,
		Name:     "file1.txt",
	}, env.scope)
	if err != nil {
		t.Fatalf("CreateFile file1: %v", err)
	}

	err = env.vfs.WriteFile(ctx, env.db, file1, virtualfs.FileContentFromReader(bytes.NewReader([]byte("content1"))), env.scope)
	if err != nil {
		t.Fatalf("WriteFile file1: %v", err)
	}

	file2, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
		ParentID: env.rootDirID,
		Name:     "file2.txt",
	}, env.scope)
	if err != nil {
		t.Fatalf("CreateFile file2: %v", err)
	}

	err = env.vfs.WriteFile(ctx, env.db, file2, virtualfs.FileContentFromReader(bytes.NewReader([]byte("content2"))), env.scope)
	if err != nil {
		t.Fatalf("WriteFile file2: %v", err)
	}

	subdir, err := env.vfs.CreateDirectory(ctx, env.db, env.rootDirID, "subdir", env.scope)
	if err != nil {
		t.Fatalf("CreateDirectory subdir: %v", err)
	}

	nestedFile, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
		ParentID: subdir.ID,
		Name:     "nested.txt",
	}, env.scope)
	if err != nil {
		t.Fatalf("CreateFile nested: %v", err)
	}

	err = env.vfs.WriteFile(ctx, env.db, nestedFile, virtualfs.FileContentFromReader(bytes.NewReader([]byte("nested content"))), env.scope)
	if err != nil {
		t.Fatalf("WriteFile nested: %v", err)
	}

	// Helper to build URLs with org slug
	driveURL := func(path string) string {
		return fmt.Sprintf("/api/%s/drives/%s%s", env.orgSlug, env.driveID, path)
	}

	t.Run("bulk trash rejects mixed kinds without deleting anything", func(t *testing.T) {
		doJSONAuth(t, env.app, http.MethodDelete, driveURL("/files?id="+file1.PublicID+","+subdir.PublicID+"&trash=true"), env.accessToken, nil, http.StatusBadRequest, nil)
		for _, node := range []*virtualfs.Node{file1, subdir} {
			if _, err := env.vfs.FindNodeByPublicID(ctx, env.db, node.PublicID, env.scope); err != nil {
				t.Fatalf("rejected bulk operation changed node: %v", err)
			}
		}
	})

	// Directory tests
	t.Run("list root directory", func(t *testing.T) {
		var resp struct {
			Items []struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL("/directories/root/content"), env.accessToken, nil, http.StatusOK, &resp)

		if len(resp.Items) != 3 {
			t.Fatalf("expected 3 items in root, got %d", len(resp.Items))
		}

		names := map[string]string{}
		for _, item := range resp.Items {
			names[item.Name] = item.Kind
		}

		if names["file1.txt"] != "file" {
			t.Errorf("expected file1.txt to be a file")
		}

		if names["file2.txt"] != "file" {
			t.Errorf("expected file2.txt to be a file")
		}

		if names["subdir"] != "directory" {
			t.Errorf("expected subdir to be a directory")
		}
	})

	t.Run("fetch directory by ID", func(t *testing.T) {
		var resp struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL(fmt.Sprintf("/directories/%s", subdir.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if resp.Kind != "directory" {
			t.Errorf("expected kind=directory, got %q", resp.Kind)
		}

		if resp.Name != "subdir" {
			t.Errorf("expected name=subdir, got %q", resp.Name)
		}

		if resp.ID != subdir.PublicID {
			t.Errorf("expected id=%s, got %s", subdir.PublicID, resp.ID)
		}
	})

	t.Run("list subdir contents", func(t *testing.T) {
		var resp struct {
			Items []struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL(fmt.Sprintf("/directories/%s/content", subdir.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if len(resp.Items) != 1 {
			t.Fatalf("expected 1 item in subdir, got %d", len(resp.Items))
		}

		if resp.Items[0].Name != "nested.txt" {
			t.Errorf("expected nested.txt, got %q", resp.Items[0].Name)
		}

		if resp.Items[0].Kind != "file" {
			t.Errorf("expected kind=file, got %q", resp.Items[0].Kind)
		}
	})

	t.Run("create directory", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		var resp struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		body := map[string]string{"parentID": root.PublicID, "name": "newdir"}
		doJSONAuth(t, env.app, http.MethodPost, driveURL("/directories"), env.accessToken, body, http.StatusOK, &resp)

		if resp.Kind != "directory" {
			t.Errorf("expected kind=directory, got %q", resp.Kind)
		}

		if resp.Name != "newdir" {
			t.Errorf("expected name=newdir, got %q", resp.Name)
		}

		if resp.ID == "" {
			t.Errorf("expected non-empty ID")
		}
	})

	t.Run("rename directory", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		renameDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "to-rename", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory to-rename: %v", err)
		}

		var resp struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		body := map[string]string{"name": "renamed-dir"}
		doJSONAuth(t, env.app, http.MethodPatch, driveURL(fmt.Sprintf("/directories/%s", renameDir.PublicID)), env.accessToken, body, http.StatusOK, &resp)

		if resp.Kind != "directory" {
			t.Errorf("expected kind=directory, got %q", resp.Kind)
		}

		if resp.Name != "renamed-dir" {
			t.Errorf("expected name=renamed-dir, got %q", resp.Name)
		}
	})

	// File tests
	t.Run("fetch file by ID", func(t *testing.T) {
		var resp struct {
			Kind     string `json:"kind"`
			ID       string `json:"id"`
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			MimeType string `json:"mimeType"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL(fmt.Sprintf("/files/%s", file1.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if resp.Kind != "file" {
			t.Errorf("expected kind=file, got %q", resp.Kind)
		}

		if resp.Name != "file1.txt" {
			t.Errorf("expected name=file1.txt, got %q", resp.Name)
		}

		if resp.Size != 8 {
			t.Errorf("expected size=8, got %d", resp.Size)
		}
	})

	t.Run("download file content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, driveURL(fmt.Sprintf("/files/%s/content", file1.PublicID)), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
		}

		content, _ := io.ReadAll(res.Body)
		if string(content) != "content1" {
			t.Errorf("expected content1, got %q", string(content))
		}
	})

	t.Run("rename file", func(t *testing.T) {
		// Create a fresh file to rename (must write content to set status to ready)
		renameFile, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "to-rename.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile to-rename: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, renameFile, virtualfs.FileContentFromReader(bytes.NewReader([]byte("rename me"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile to-rename: %v", err)
		}

		var resp struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		body := map[string]string{"name": "renamed-file.txt"}
		doJSONAuth(t, env.app, http.MethodPatch, driveURL(fmt.Sprintf("/files/%s", renameFile.PublicID)), env.accessToken, body, http.StatusOK, &resp)

		if resp.Name != "renamed-file.txt" {
			t.Errorf("expected name=renamed-file.txt, got %q", resp.Name)
		}
	})

	t.Run("fetch nested file", func(t *testing.T) {
		var resp struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Name string `json:"name"`
			Size int64  `json:"size"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL(fmt.Sprintf("/files/%s", nestedFile.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if resp.Name != "nested.txt" {
			t.Errorf("expected name=nested.txt, got %q", resp.Name)
		}

		if resp.Size != 14 {
			t.Errorf("expected size=14, got %d", resp.Size)
		}
	})

	t.Run("download nested file content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, driveURL(fmt.Sprintf("/files/%s/content", nestedFile.PublicID)), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
		}

		content, _ := io.ReadAll(res.Body)
		if string(content) != "nested content" {
			t.Errorf("expected 'nested content', got %q", string(content))
		}
	})

	// Delete tests
	t.Run("trash file", func(t *testing.T) {
		// Create a fresh file to trash (must write content to set status to ready)
		trashFile, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "to-trash.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile to-trash: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, trashFile, virtualfs.FileContentFromReader(bytes.NewReader([]byte("trash me"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile to-trash: %v", err)
		}

		var resp struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Kind string `json:"kind"`
		}

		doJSONAuth(t, env.app, http.MethodDelete, driveURL(fmt.Sprintf("/files/%s?trash=true", trashFile.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if resp.ID != trashFile.PublicID {
			t.Errorf("expected id=%s, got %s", trashFile.PublicID, resp.ID)
		}

		if resp.Kind != "file" {
			t.Errorf("expected kind=file, got %q", resp.Kind)
		}
	})

	t.Run("delete file permanently", func(t *testing.T) {
		// Create a temp file to delete (must write content to set status to ready)
		tempFile, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "temp.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile temp: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, tempFile, virtualfs.FileContentFromReader(bytes.NewReader([]byte("delete me"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile temp: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, driveURL(fmt.Sprintf("/files/%s", tempFile.PublicID)), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", res.StatusCode)
		}

		// Verify file is gone
		req = httptest.NewRequest(http.MethodGet, driveURL(fmt.Sprintf("/files/%s", tempFile.PublicID)), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err = env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 after permanent delete, got %d", res.StatusCode)
		}
	})

	t.Run("404 for non-existent file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, driveURL("/files/nonexistent123"), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", res.StatusCode)
		}
	})

	t.Run("404 for non-existent directory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, driveURL("/directories/nonexistent123"), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", res.StatusCode)
		}
	})

	// Bulk delete tests
	t.Run("bulk trash files", func(t *testing.T) {
		// Create temp files for bulk delete (must write content to set status to ready)
		bulkFile1, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "bulk1.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile bulk1: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, bulkFile1, virtualfs.FileContentFromReader(bytes.NewReader([]byte("bulk1"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile bulk1: %v", err)
		}

		bulkFile2, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "bulk2.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile bulk2: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, bulkFile2, virtualfs.FileContentFromReader(bytes.NewReader([]byte("bulk2"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile bulk2: %v", err)
		}

		var resp []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Kind string `json:"kind"`
		}

		url := driveURL(fmt.Sprintf("/files?id=%s,%s&trash=true", bulkFile1.PublicID, bulkFile2.PublicID))
		doJSONAuth(t, env.app, http.MethodDelete, url, env.accessToken, nil, http.StatusOK, &resp)

		if len(resp) != 2 {
			t.Fatalf("expected 2 trashed files, got %d", len(resp))
		}

		for _, f := range resp {
			if f.Kind != "file" {
				t.Errorf("expected kind=file for %s, got %q", f.Name, f.Kind)
			}
		}
	})

	t.Run("bulk delete files permanently", func(t *testing.T) {
		// Create temp files for bulk delete (must write content to set status to ready)
		bulkFile3, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "bulk3.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile bulk3: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, bulkFile3, virtualfs.FileContentFromReader(bytes.NewReader([]byte("bulk3"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile bulk3: %v", err)
		}

		bulkFile4, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "bulk4.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile bulk4: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, bulkFile4, virtualfs.FileContentFromReader(bytes.NewReader([]byte("bulk4"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile bulk4: %v", err)
		}

		url := driveURL(fmt.Sprintf("/files?id=%s,%s", bulkFile3.PublicID, bulkFile4.PublicID))
		req := httptest.NewRequest(http.MethodDelete, url, nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", res.StatusCode)
		}

		// Verify files are gone
		for _, fileID := range []string{bulkFile3.PublicID, bulkFile4.PublicID} {
			req = httptest.NewRequest(http.MethodGet, driveURL(fmt.Sprintf("/files/%s", fileID)), nil)
			req.Header.Set("Authorization", "Bearer "+env.accessToken)

			res, err = env.app.Test(req, 10_000)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			res.Body.Close()

			if res.StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 for %s after permanent delete, got %d", fileID, res.StatusCode)
			}
		}
	})

	t.Run("bulk trash directories", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create temp directories for bulk delete
		bulkDir1, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "bulkdir1", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory bulkdir1: %v", err)
		}

		bulkDir2, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "bulkdir2", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory bulkdir2: %v", err)
		}

		var resp []struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			DeletedAt *string `json:"deletedAt"`
		}

		url := driveURL(fmt.Sprintf("/directories?id=%s,%s&trash=true", bulkDir1.PublicID, bulkDir2.PublicID))
		doJSONAuth(t, env.app, http.MethodDelete, url, env.accessToken, nil, http.StatusOK, &resp)

		if len(resp) != 2 {
			t.Fatalf("expected 2 trashed directories, got %d", len(resp))
		}

		for _, d := range resp {
			if d.DeletedAt == nil {
				t.Errorf("expected deletedAt to be set for %s", d.Name)
			}
		}
	})

	t.Run("bulk delete directories permanently", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create temp directories for bulk delete
		bulkDir3, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "bulkdir3", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory bulkdir3: %v", err)
		}

		bulkDir4, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "bulkdir4", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory bulkdir4: %v", err)
		}

		url := driveURL(fmt.Sprintf("/directories?id=%s,%s", bulkDir3.PublicID, bulkDir4.PublicID))
		req := httptest.NewRequest(http.MethodDelete, url, nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", res.StatusCode)
		}

		// Verify directories are gone
		for _, dirID := range []string{bulkDir3.PublicID, bulkDir4.PublicID} {
			req = httptest.NewRequest(http.MethodGet, driveURL(fmt.Sprintf("/directories/%s", dirID)), nil)
			req.Header.Set("Authorization", "Bearer "+env.accessToken)

			res, err = env.app.Test(req, 10_000)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			res.Body.Close()

			if res.StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 for %s after permanent delete, got %d", dirID, res.StatusCode)
			}
		}
	})

	t.Run("delete directory with trash", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		trashDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "trashme", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory trashme: %v", err)
		}

		var resp struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			DeletedAt *string `json:"deletedAt"`
		}

		doJSONAuth(t, env.app, http.MethodDelete, driveURL(fmt.Sprintf("/directories/%s?trash=true", trashDir.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if resp.DeletedAt == nil {
			t.Errorf("expected deletedAt to be set")
		}
	})

	t.Run("delete directory permanently", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		permDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "deleteme", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory deleteme: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, driveURL(fmt.Sprintf("/directories/%s", permDir.PublicID)), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err := env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", res.StatusCode)
		}

		// Verify directory is gone
		req = httptest.NewRequest(http.MethodGet, driveURL(fmt.Sprintf("/directories/%s", permDir.PublicID)), nil)
		req.Header.Set("Authorization", "Bearer "+env.accessToken)

		res, err = env.app.Test(req, 10_000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 after permanent delete, got %d", res.StatusCode)
		}
	})

	// Move items tests
	t.Run("move files to directory", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create source files in root
		moveFile1, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: root.ID,
			Name:     "move1.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile move1: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, moveFile1, virtualfs.FileContentFromReader(bytes.NewReader([]byte("move1"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile move1: %v", err)
		}

		moveFile2, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: root.ID,
			Name:     "move2.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile move2: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, moveFile2, virtualfs.FileContentFromReader(bytes.NewReader([]byte("move2"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile move2: %v", err)
		}

		// Create target directory
		targetDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "move-target", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory move-target: %v", err)
		}

		var resp struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			Moved     []string `json:"moved"`
			Conflicts []string `json:"conflicts"`
		}

		body := map[string][]string{"items": {moveFile1.PublicID, moveFile2.PublicID}}
		doJSONAuth(t, env.app, http.MethodPost, driveURL(fmt.Sprintf("/directories/%s/content", targetDir.PublicID)), env.accessToken, body, http.StatusOK, &resp)

		if len(resp.Moved) != 2 {
			t.Errorf("expected 2 moved items, got %d", len(resp.Moved))
		}

		if len(resp.Conflicts) != 0 {
			t.Errorf("expected 0 conflicts, got %d", len(resp.Conflicts))
		}
	})

	t.Run("move items from different directories fails", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create file in root
		fileInRoot, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: root.ID,
			Name:     "in-root.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile in-root: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, fileInRoot, virtualfs.FileContentFromReader(bytes.NewReader([]byte("root"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile in-root: %v", err)
		}

		// Create subdirectory with file
		subDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "subdir-for-move", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory subdir-for-move: %v", err)
		}

		fileInSub, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: subDir.ID,
			Name:     "in-sub.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile in-sub: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, fileInSub, virtualfs.FileContentFromReader(bytes.NewReader([]byte("sub"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile in-sub: %v", err)
		}

		// Create target directory
		targetDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "target-for-fail", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory target-for-fail: %v", err)
		}

		// Try to move files from different directories
		var resp struct {
			Error string `json:"error"`
		}

		body := map[string][]string{"items": {fileInRoot.PublicID, fileInSub.PublicID}}
		doJSONAuth(t, env.app, http.MethodPost, driveURL(fmt.Sprintf("/directories/%s/content", targetDir.PublicID)), env.accessToken, body, http.StatusBadRequest, &resp)

		if resp.Error != "All items must be in the same directory" {
			t.Errorf("expected 'All items must be in the same directory', got %q", resp.Error)
		}
	})

	t.Run("move items with name conflict", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create source file
		srcFile, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: root.ID,
			Name:     "conflict-file.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile conflict-file: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, srcFile, virtualfs.FileContentFromReader(bytes.NewReader([]byte("src"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile conflict-file: %v", err)
		}

		// Create target directory with file of same name
		targetDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "conflict-target", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory conflict-target: %v", err)
		}

		existingFile, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: targetDir.ID,
			Name:     "conflict-file.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile existing: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, existingFile, virtualfs.FileContentFromReader(bytes.NewReader([]byte("existing"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile existing: %v", err)
		}

		// Move should report conflict
		var resp struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			Moved     []string `json:"moved"`
			Conflicts []string `json:"conflicts"`
		}

		body := map[string][]string{"items": {srcFile.PublicID}}
		doJSONAuth(t, env.app, http.MethodPost, driveURL(fmt.Sprintf("/directories/%s/content", targetDir.PublicID)), env.accessToken, body, http.StatusOK, &resp)

		if len(resp.Conflicts) != 1 {
			t.Errorf("expected 1 conflict, got %d", len(resp.Conflicts))
		}

		if len(resp.Moved) != 0 {
			t.Errorf("expected 0 moved, got %d", len(resp.Moved))
		}
	})

	t.Run("move items not found", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		targetDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "target-notfound", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory target-notfound: %v", err)
		}

		var resp struct {
			Error string `json:"error"`
		}

		body := map[string][]string{"items": {"nonexistent123", "nonexistent456"}}
		doJSONAuth(t, env.app, http.MethodPost, driveURL(fmt.Sprintf("/directories/%s/content", targetDir.PublicID)), env.accessToken, body, http.StatusNotFound, &resp)

		if resp.Error != "One or more items not found" {
			t.Errorf("expected 'One or more items not found', got %q", resp.Error)
		}
	})

	// Error cases for createDirectory
	t.Run("create directory with nonexistent parent", func(t *testing.T) {
		var resp struct {
			Error string `json:"error"`
		}

		body := map[string]string{"parentID": "nonexistent123", "name": "orphan"}
		doJSONAuth(t, env.app, http.MethodPost, driveURL("/directories"), env.accessToken, body, http.StatusBadRequest, &resp)

		if resp.Error != "Parent not found" {
			t.Errorf("expected 'Parent not found', got %q", resp.Error)
		}
	})

	t.Run("create directory with file as parent", func(t *testing.T) {
		var resp struct {
			Error string `json:"error"`
		}

		body := map[string]string{"parentID": file1.PublicID, "name": "under-file"}
		doJSONAuth(t, env.app, http.MethodPost, driveURL("/directories"), env.accessToken, body, http.StatusBadRequest, &resp)

		if resp.Error != "Parent is not a directory" {
			t.Errorf("expected 'Parent is not a directory', got %q", resp.Error)
		}
	})

	t.Run("create directory with duplicate name", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create first directory
		_, err = env.vfs.CreateDirectory(ctx, env.db, root.ID, "duplicate-dir", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory duplicate-dir: %v", err)
		}

		// Try to create another with same name
		var resp struct {
			Error string `json:"error"`
		}

		body := map[string]string{"parentID": root.PublicID, "name": "duplicate-dir"}
		doJSONAuth(t, env.app, http.MethodPost, driveURL("/directories"), env.accessToken, body, http.StatusConflict, &resp)

		if resp.Error != "Directory already exists" {
			t.Errorf("expected 'Directory already exists', got %q", resp.Error)
		}
	})

	// Error cases for bulk delete
	t.Run("bulk delete files rejects directories", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		dirToDelete, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "dir-in-files-delete", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory: %v", err)
		}

		var resp struct {
			Error string `json:"error"`
		}

		url := driveURL(fmt.Sprintf("/files?id=%s", dirToDelete.PublicID))
		doJSONAuth(t, env.app, http.MethodDelete, url, env.accessToken, nil, http.StatusBadRequest, &resp)

		if resp.Error != "all items must be files" {
			t.Errorf("expected 'all items must be files', got %q", resp.Error)
		}
	})

	t.Run("bulk delete directories rejects files", func(t *testing.T) {
		fileToDelete, err := env.vfs.CreateFile(ctx, env.db, virtualfs.CreateFileOptions{
			ParentID: env.rootDirID,
			Name:     "file-in-dirs-delete.txt",
		}, env.scope)
		if err != nil {
			t.Fatalf("CreateFile: %v", err)
		}

		err = env.vfs.WriteFile(ctx, env.db, fileToDelete, virtualfs.FileContentFromReader(bytes.NewReader([]byte("x"))), env.scope)
		if err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		var resp struct {
			Error string `json:"error"`
		}

		url := driveURL(fmt.Sprintf("/directories?id=%s", fileToDelete.PublicID))
		doJSONAuth(t, env.app, http.MethodDelete, url, env.accessToken, nil, http.StatusBadRequest, &resp)

		if resp.Error != "all items must be directories" {
			t.Errorf("expected 'all items must be directories', got %q", resp.Error)
		}
	})

	// Pagination tests
	t.Run("list directory with limit", func(t *testing.T) {
		root, err := env.vfs.FindNode(ctx, env.db, env.rootDirID.String(), env.scope)
		if err != nil {
			t.Fatalf("FindNode root: %v", err)
		}

		// Create a directory with multiple items
		paginationDir, err := env.vfs.CreateDirectory(ctx, env.db, root.ID, "pagination-test", env.scope)
		if err != nil {
			t.Fatalf("CreateDirectory pagination-test: %v", err)
		}

		for i := 0; i < 5; i++ {
			_, err := env.vfs.CreateDirectory(ctx, env.db, paginationDir.ID, fmt.Sprintf("item-%d", i), env.scope)
			if err != nil {
				t.Fatalf("CreateDirectory item-%d: %v", i, err)
			}
		}

		var resp struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			NextCursor string `json:"nextCursor"`
		}

		// Must specify orderBy to match cursor encoding
		doJSONAuth(t, env.app, http.MethodGet, driveURL(fmt.Sprintf("/directories/%s/content?limit=2&orderBy=name", paginationDir.PublicID)), env.accessToken, nil, http.StatusOK, &resp)

		if len(resp.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(resp.Items))
		}

		if resp.NextCursor == "" {
			t.Errorf("expected nextCursor to be set")
		}

		// Fetch next page - must use same orderBy
		var resp2 struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			NextCursor string `json:"nextCursor"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL(fmt.Sprintf("/directories/%s/content?limit=2&orderBy=name&cursor=%s", paginationDir.PublicID, resp.NextCursor)), env.accessToken, nil, http.StatusOK, &resp2)

		if len(resp2.Items) != 2 {
			t.Errorf("expected 2 items on second page, got %d", len(resp2.Items))
		}
	})

	t.Run("list directory with invalid limit", func(t *testing.T) {
		var resp struct {
			Error string `json:"error"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL("/directories/root/content?limit=0"), env.accessToken, nil, http.StatusBadRequest, &resp)

		if resp.Error != "Limit must be at least 1" {
			t.Errorf("expected 'Limit must be at least 1', got %q", resp.Error)
		}
	})

	t.Run("list directory with invalid cursor", func(t *testing.T) {
		var resp struct {
			Error string `json:"error"`
		}

		doJSONAuth(t, env.app, http.MethodGet, driveURL("/directories/root/content?cursor=invalid"), env.accessToken, nil, http.StatusBadRequest, &resp)

		if resp.Error != "invalid cursor" {
			t.Errorf("expected 'invalid cursor', got %q", resp.Error)
		}
	})
}

func doJSONAuth(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
	accessToken string,
	body any,
	wantStatus int,
	out any,
) {
	t.Helper()

	var reqBody *bytes.Reader

	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json marshal: %v", err)
		}

		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	res, err := app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}

	defer res.Body.Close()

	if res.StatusCode != wantStatus {
		gotBody, _ := io.ReadAll(res.Body)
		t.Fatalf("%s %s: status %d want %d body=%s", method, path, res.StatusCode, wantStatus, string(gotBody))
	}

	if out == nil {
		return
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("%s %s: decode response: %v", method, path, err)
	}
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
