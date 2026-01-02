package drexa

import (
	"context"
	"fmt"
	"strings"

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
	"github.com/get-drexa/drexa/internal/upload"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Configure CORS middleware
	corsConfig := cors.Config{
		AllowOrigins:     "",
		AllowCredentials: c.CORS.AllowCredentials,
	}
	if len(c.CORS.AllowOrigins) > 0 {
		corsConfig.AllowOrigins = strings.Join(c.CORS.AllowOrigins, ",")
	}
	app.Use(cors.New(corsConfig))

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

	sharingService, err := sharing.NewService(vfs)
	if err != nil {
		return nil, fmt.Errorf("failed to create sharing service: %w", err)
	}

	userService := user.NewService()
	organizationService := organization.NewService()
	accountService := account.NewService()
	driveService := drive.NewService()
	authService := auth.NewService(userService, auth.TokenConfig{
		Issuer:    c.JWT.Issuer,
		Audience:  c.JWT.Audience,
		SecretKey: c.JWT.SecretKey,
	})
	uploadService := upload.NewService(vfs, blobStore)
	registrationService := registration.NewService(userService, organizationService, accountService, driveService, vfs)

	cookieConfig := auth.CookieConfig{
		Domain: c.Cookie.Domain,
	}

	authMiddleware := auth.NewAuthMiddleware(authService, db, cookieConfig)
	optionalAuthMiddleware := auth.NewOptionalAuthMiddleware(authService, db, cookieConfig)

	api := app.Group("/api")
	auth.NewHTTPHandler(authService, organizationService, db, cookieConfig).RegisterRoutes(api)
	registration.NewHTTPHandler(registrationService, authService, db, cookieConfig).RegisterRoutes(api)
	user.NewHTTPHandler(userService, db, authMiddleware).RegisterRoutes(api)
	account.NewHTTPHandler(accountService, db, authMiddleware).RegisterRoutes(api)

	orgAPI := api.Group("/:orgSlug", authMiddleware, organization.NewMiddleware(organizationService, accountService, db))

	driveRouter := drive.NewHTTPHandler(driveService, accountService, vfs, db, authMiddleware).RegisterRoutes(orgAPI)
	upload.NewHTTPHandler(uploadService, db).RegisterRoutes(driveRouter)

	shareHTTP := sharing.NewHTTPHandler(sharingService, accountService, driveService, vfs, db, optionalAuthMiddleware)
	shareRoutes := shareHTTP.RegisterShareConsumeRoutes(api)
	shareHTTP.RegisterShareManagementRoutes(driveRouter)

	catalogHTTP := catalog.NewHTTPHandler(sharingService, vfs, db)
	catalogHTTP.RegisterRoutes(driveRouter)
	catalogHTTP.RegisterRoutes(shareRoutes)

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
