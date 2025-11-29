package drexa

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/get-drexa/drexa/internal/auth"
	"github.com/get-drexa/drexa/internal/blob"
	"github.com/get-drexa/drexa/internal/database"
	"github.com/get-drexa/drexa/internal/upload"
	"github.com/get-drexa/drexa/internal/user"
	"github.com/get-drexa/drexa/internal/virtualfs"
	"github.com/gofiber/fiber/v2"
)

type ServerConfig struct {
	Port         int
	PostgresURL  string
	JWTIssuer    string
	JWTAudience  string
	JWTSecretKey []byte
}

func NewServer(c ServerConfig) (*fiber.App, error) {
	app := fiber.New()
	db := database.NewFromPostgres(c.PostgresURL)

	// TODO: load correct blob store and resolver from config
	blobStore := blob.NewFSStore(blob.FSStoreConfig{
		Root:      os.Getenv("BLOB_ROOT"),
		UploadURL: os.Getenv("BLOB_UPLOAD_URL"),
	})
	keyResolver := virtualfs.NewFlatKeyResolver()
	vfs, err := virtualfs.NewVirtualFS(db, blobStore, keyResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual file system: %w", err)
	}

	userService := user.NewService(db)
	authService := auth.NewService(db, userService, auth.TokenConfig{
		Issuer:    c.JWTIssuer,
		Audience:  c.JWTAudience,
		SecretKey: c.JWTSecretKey,
	})
	uploadService := upload.NewService(vfs, blobStore)

	api := app.Group("/api")
	auth.RegisterAPIRoutes(api, authService)
	upload.RegisterAPIRoutes(api, uploadService)

	return app, nil
}

// ServerConfigFromEnv creates a ServerConfig from environment variables.
func ServerConfigFromEnv() (*ServerConfig, error) {
	c := ServerConfig{
		PostgresURL: os.Getenv("POSTGRES_URL"),
		JWTIssuer:   os.Getenv("JWT_ISSUER"),
		JWTAudience: os.Getenv("JWT_AUDIENCE"),
	}

	errs := []error{}

	keyHex := os.Getenv("JWT_SECRET_KEY")
	if keyHex == "" {
		errs = append(errs, errors.New("JWT_SECRET_KEY is required"))
	} else {
		k, err := hex.DecodeString(keyHex)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to decode JWT_SECRET_KEY: %w", err))
		}
		c.JWTSecretKey = k
	}

	p, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to parse PORT: %w", err))
	}
	c.Port = p

	if c.PostgresURL == "" {
		errs = append(errs, errors.New("POSTGRES_URL is required"))
	}
	if c.JWTIssuer == "" {
		errs = append(errs, errors.New("JWT_ISSUER is required"))
	}
	if c.JWTAudience == "" {
		errs = append(errs, errors.New("JWT_AUDIENCE is required"))
	}
	if len(c.JWTSecretKey) == 0 {
		errs = append(errs, errors.New("JWT_SECRET_KEY is required"))
	}

	if len(errs) > 0 {
		return nil, NewServerConfigError(errs...)
	}

	return &c, nil
}
