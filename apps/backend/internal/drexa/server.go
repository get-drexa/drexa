package drexa

import (
	"context"
	"fmt"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/upload"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func NewServer(c Config) (*fiber.App, error) {
	app := fiber.New()
	db := database.NewFromPostgres(c.Database.PostgresURL)

	app.Use(logger.New())

	// Initialize blob store based on config
	var blobStore blob.Store
	switch c.Storage.Backend {
	case StorageBackendFS:
		blobStore = blob.NewFSStore(blob.FSStoreConfig{
			Root: c.Storage.RootPath,
		})
	case StorageBackendS3:
		return nil, fmt.Errorf("s3 storage backend not yet implemented")
	default:
		return nil, fmt.Errorf("unknown storage backend: %s", c.Storage.Backend)
	}

	err := blobStore.Initialize(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blob store: %w", err)
	}

	// Initialize key resolver based on config
	var keyResolver virtualfs.BlobKeyResolver
	switch c.Storage.Mode {
	case StorageModeFlat:
		keyResolver = virtualfs.NewFlatKeyResolver()
	case StorageModeHierarchical:
		keyResolver = virtualfs.NewHierarchicalKeyResolver(db)
	default:
		return nil, fmt.Errorf("unknown storage mode: %s", c.Storage.Mode)
	}

	vfs, err := virtualfs.NewVirtualFS(blobStore, keyResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual file system: %w", err)
	}

	userService := user.NewService()
	authService := auth.NewService(userService, auth.TokenConfig{
		Issuer:    c.JWT.Issuer,
		Audience:  c.JWT.Audience,
		SecretKey: c.JWT.SecretKey,
	})
	uploadService := upload.NewService(vfs, blobStore)
	accountService := account.NewService(userService, vfs)

	authMiddleware := auth.NewBearerAuthMiddleware(authService, db)

	api := app.Group("/api")

	accRouter := account.NewHTTPHandler(accountService, authService, db, authMiddleware).RegisterRoutes(api)

	auth.NewHTTPHandler(authService, db).RegisterRoutes(api)
	upload.NewHTTPHandler(uploadService, db).RegisterRoutes(accRouter)

	return app, nil
}
