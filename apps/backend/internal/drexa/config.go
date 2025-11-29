package drexa

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type StorageMode string
type StorageBackend string

const (
	StorageModeFlat         StorageMode = "flat"
	StorageModeHierarchical StorageMode = "hierarchical"
)

const (
	StorageBackendFS StorageBackend = "fs"
	StorageBackendS3 StorageBackend = "s3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Storage  StorageConfig  `yaml:"storage"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	PostgresURL string `yaml:"postgres_url"`
}

type JWTConfig struct {
	Issuer          string `yaml:"issuer"`
	Audience        string `yaml:"audience"`
	SecretKeyBase64 string `yaml:"secret_key_base64"`
	SecretKeyPath   string `yaml:"secret_key_path"`
	SecretKey       []byte `yaml:"-"`
}

type StorageConfig struct {
	Mode     StorageMode    `yaml:"mode"`
	Backend  StorageBackend `yaml:"backend"`
	RootPath string         `yaml:"root_path"`
	Bucket   string         `yaml:"bucket"`
}

// ConfigFromFile loads configuration from a YAML file.
// JWT secret key is loaded from JWT_SECRET_KEY env var (base64 encoded),
// falling back to the file path specified in jwt.secret_key_path.
func ConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Load JWT secret key (priority: env var > config base64 > config file path)
	if envKey := os.Getenv("JWT_SECRET_KEY"); envKey != "" {
		key, err := base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			return nil, errors.New("JWT_SECRET_KEY env var is not valid base64")
		}
		config.JWT.SecretKey = key
	} else if config.JWT.SecretKeyBase64 != "" {
		key, err := base64.StdEncoding.DecodeString(config.JWT.SecretKeyBase64)
		if err != nil {
			return nil, errors.New("jwt.secret_key_base64 is not valid base64")
		}
		config.JWT.SecretKey = key
	} else if config.JWT.SecretKeyPath != "" {
		keyData, err := os.ReadFile(config.JWT.SecretKeyPath)
		if err != nil {
			return nil, err
		}
		key, err := base64.StdEncoding.DecodeString(string(keyData))
		if err != nil {
			return nil, errors.New("jwt.secret_key_path file content is not valid base64")
		}
		config.JWT.SecretKey = key
	}

	if errs := config.Validate(); len(errs) > 0 {
		return nil, NewConfigError(errs...)
	}

	return &config, nil
}

// Validate checks for required configuration fields.
func (c *Config) Validate() []error {
	var errs []error

	// Server
	if c.Server.Port == 0 {
		errs = append(errs, errors.New("server.port is required"))
	}

	// Database
	if c.Database.PostgresURL == "" {
		errs = append(errs, errors.New("database.postgres_url is required"))
	}

	// JWT
	if c.JWT.Issuer == "" {
		errs = append(errs, errors.New("jwt.issuer is required"))
	}
	if c.JWT.Audience == "" {
		errs = append(errs, errors.New("jwt.audience is required"))
	}
	if len(c.JWT.SecretKey) == 0 {
		errs = append(errs, errors.New("jwt secret key is required (set JWT_SECRET_KEY env var, jwt.secret_key_base64, or jwt.secret_key_path)"))
	}

	// Storage
	if c.Storage.Mode == "" {
		errs = append(errs, errors.New("storage.mode is required"))
	} else if c.Storage.Mode != StorageModeFlat && c.Storage.Mode != StorageModeHierarchical {
		errs = append(errs, errors.New("storage.mode must be 'flat' or 'hierarchical'"))
	}

	if c.Storage.Backend == "" {
		errs = append(errs, errors.New("storage.backend is required"))
	} else if c.Storage.Backend != StorageBackendFS && c.Storage.Backend != StorageBackendS3 {
		errs = append(errs, errors.New("storage.backend must be 'fs' or 's3'"))
	}

	if c.Storage.Backend == StorageBackendFS && c.Storage.RootPath == "" {
		errs = append(errs, errors.New("storage.root_path is required when backend is 'fs'"))
	}
	if c.Storage.Backend == StorageBackendS3 {
		if c.Storage.Bucket == "" {
			errs = append(errs, errors.New("storage.bucket is required when backend is 's3'"))
		}
		if c.Storage.Mode == StorageModeHierarchical {
			errs = append(errs, errors.New("storage.mode must be 'flat' when backend is 's3'"))
		}
	}

	return errs
}
