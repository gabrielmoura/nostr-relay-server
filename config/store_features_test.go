package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateStoreFeatures(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "local backend",
			cfg:  &Config{Store: StoreConfig{Backend: "local"}},
		},
		{
			name:    "unknown backend",
			cfg:     &Config{Store: StoreConfig{Backend: "memory"}},
			wantErr: "store.backend must be one of: local, s3",
		},
		{
			name: "s3 requires endpoint",
			cfg: &Config{Store: StoreConfig{
				Backend: "s3",
				S3:      StoreS3Config{Bucket: "blobs", PresignTTL: time.Minute},
			}},
			wantErr: "store.s3.endpoint is required when store.backend is s3",
		},
		{
			name: "s3 requires credentials",
			cfg: &Config{Store: StoreConfig{
				Backend: "s3",
				S3: StoreS3Config{
					Endpoint:   "http://localhost:9000",
					Bucket:     "blobs",
					PresignTTL: time.Minute,
				},
			}},
			wantErr: "store.s3.access_key and store.s3.secret_key are required when store.backend is s3",
		},
		{
			name: "s3 requires both credentials",
			cfg: &Config{Store: StoreConfig{
				Backend: "s3",
				S3: StoreS3Config{
					Endpoint:   "http://localhost:9000",
					Bucket:     "blobs",
					AccessKey:  "access",
					PresignTTL: time.Minute,
				},
			}},
			wantErr: "store.s3.access_key and store.s3.secret_key are required when store.backend is s3",
		},
		{
			name: "valid s3 configuration",
			cfg: &Config{Store: StoreConfig{
				Backend: "s3",
				S3: StoreS3Config{
					Endpoint:   "http://localhost:9000",
					Bucket:     "blobs",
					AccessKey:  "access",
					SecretKey:  "secret",
					PresignTTL: time.Minute,
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateStoreFeatures()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ValidateStoreFeatures() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("ValidateStoreFeatures() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestApplyStoreS3EnvironmentOverridesConfiguration(t *testing.T) {
	t.Setenv("NRS_STORE_S3_ACCESS_KEY", "environment-access")
	t.Setenv("NRS_STORE_S3_SECRET_KEY", "environment-secret")

	cfg := &Config{Store: StoreConfig{S3: StoreS3Config{
		AccessKey: "file-access",
		SecretKey: "file-secret",
	}}}
	applyStoreS3Environment(cfg)

	if cfg.Store.S3.AccessKey != "environment-access" || cfg.Store.S3.SecretKey != "environment-secret" {
		t.Fatalf("environment credentials were not applied: %#v", cfg.Store.S3)
	}

	redacted := cfg.Redacted()
	if redacted.Store.S3.AccessKey != "" || redacted.Store.S3.SecretKey != "" {
		t.Fatalf("Redacted() leaked credentials: %#v", redacted.Store.S3)
	}
}

func TestWriteYamlConfigRedactsStoreS3EnvironmentCredentials(t *testing.T) {
	t.Setenv("NRS_STORE_S3_ACCESS_KEY", "environment-access")
	t.Setenv("NRS_STORE_S3_SECRET_KEY", "environment-secret")

	path := filepath.Join(t.TempDir(), "conf.yaml")
	if err := WriteYamlConfig(path); err != nil {
		t.Fatalf("WriteYamlConfig() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	output := string(data)
	if strings.Contains(output, "environment-access") || strings.Contains(output, "environment-secret") {
		t.Fatalf("WriteYamlConfig() leaked credentials: %s", output)
	}
	if !strings.Contains(output, "backend: local") || !strings.Contains(output, "presign_ttl: 5m") {
		t.Fatalf("WriteYamlConfig() did not include S3 defaults: %s", output)
	}
}
