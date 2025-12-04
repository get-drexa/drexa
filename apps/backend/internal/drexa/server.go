package drexa

import (
	"context"
	"fmt"

	"github.com/get-drexa/drexa/internal/account"
	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/catalog"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/httperr"
	"github.com/get-drexa/drexa/internal/upload"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
)

type Server struct {
	config         Config
	app            *fiber.App
	db             *bun.DB
	blobStore      blob.Store
	vfs            *virtualfs.VirtualFS
	userService    *user.Service
	authService    *auth.Service
	accountService *account.Service
	uploadService  *upload.Service
	authMiddleware fiber.Handler
}

func NewServer(c Config) (*Server, error) {
	app := fiber.New(fiber.Config{
		ErrorHandler:      httperr.ErrorHandler,
		StreamRequestBody: true,
		// Trust proxy headers (X-Forwarded-Proto, X-Forwarded-For) for proper
		// protocol detection behind reverse proxies, tunnels (ngrok, cloudflare), etc.
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		ProxyHeader:             fiber.HeaderXForwardedFor,
	})
	app.Use(logger.New())

	db := database.NewFromPostgres(c.Database.PostgresURL)
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

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

	err := blobStore.Initialize(context.TODO())
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

	vfs, err := virtualfs.New(blobStore, keyResolver)
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

	cookieConfig := auth.CookieConfig{
		Domain: c.Cookie.Domain,
	}

	authMiddleware := auth.NewAuthMiddleware(authService, db, cookieConfig)

	api := app.Group("/api")
	auth.NewHTTPHandler(authService, db, cookieConfig).RegisterRoutes(api)
	user.NewHTTPHandler(userService, db, authMiddleware).RegisterRoutes(api)

	accountRouter := account.NewHTTPHandler(accountService, authService, db, authMiddleware).RegisterRoutes(api)
	upload.NewHTTPHandler(uploadService, db).RegisterRoutes(accountRouter)
	catalog.NewHTTPHandler(vfs, db).RegisterRoutes(accountRouter)

	s := &Server{
		config:         c,
		app:            app,
		db:             db,
		blobStore:      blobStore,
		vfs:            vfs,
		userService:    userService,
		authService:    authService,
		accountService: accountService,
		uploadService:  uploadService,
		authMiddleware: authMiddleware,
	}

	return s, nil
}

func (s *Server) Start() error {
	return s.app.Listen(fmt.Sprintf(":%d", s.config.Server.Port))
}
