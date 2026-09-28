package config

import (
	"fmt"
	"strings"
)

func (cfg *Config) ValidateStoreFeatures() error {
	if cfg == nil {
		return nil
	}

	switch cfg.Store.NormalizedBackend() {
	case "local":
		return nil
	case "s3":
		return validateStoreS3Config(cfg.Store.S3)
	default:
		return fmt.Errorf("store.backend must be one of: local, s3")
	}
}

func (cfg StoreConfig) NormalizedBackend() string {
	return strings.ToLower(strings.TrimSpace(cfg.Backend))
}

func validateStoreS3Config(cfg StoreS3Config) error {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return fmt.Errorf("store.s3.endpoint is required when store.backend is s3")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return fmt.Errorf("store.s3.bucket is required when store.backend is s3")
	}

	hasAccessKey := strings.TrimSpace(cfg.AccessKey) != ""
	hasSecretKey := strings.TrimSpace(cfg.SecretKey) != ""
	if !hasAccessKey || !hasSecretKey {
		return fmt.Errorf("store.s3.access_key and store.s3.secret_key are required when store.backend is s3")
	}
	if cfg.PresignTTL <= 0 {
		return fmt.Errorf("store.s3.presign_ttl must be greater than zero")
	}

	return nil
}
