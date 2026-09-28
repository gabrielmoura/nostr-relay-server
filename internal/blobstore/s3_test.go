package blobstore

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

func TestParseS3Endpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHost string
		wantTLS  bool
		wantErr  bool
	}{
		{
			name:     "HTTP",
			endpoint: "http://127.0.0.1:9000",
			wantHost: "127.0.0.1:9000",
		},
		{
			name:     "HTTPS",
			endpoint: "https://s3.example.test",
			wantHost: "s3.example.test",
			wantTLS:  true,
		},
		{
			name:     "missing scheme",
			endpoint: "s3.example.test",
			wantErr:  true,
		},
		{
			name:     "path is rejected",
			endpoint: "https://s3.example.test/storage",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, secure, err := parseS3Endpoint(tt.endpoint)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseS3Endpoint() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseS3Endpoint() error = %v", err)
			}
			if host != tt.wantHost || secure != tt.wantTLS {
				t.Fatalf("parseS3Endpoint() = %q, %t; want %q, %t", host, secure, tt.wantHost, tt.wantTLS)
			}
		})
	}
}

func TestS3StoreContract(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	accessKey := os.Getenv("TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("TEST_S3_SECRET_KEY")
	bucket := os.Getenv("TEST_S3_BUCKET")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("set TEST_S3_ENDPOINT, TEST_S3_ACCESS_KEY, TEST_S3_SECRET_KEY, and TEST_S3_BUCKET to run S3 contract tests")
	}

	store, err := NewS3Store(context.Background(), config.StoreS3Config{
		Endpoint:     endpoint,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		Bucket:       bucket,
		UsePathStyle: true,
		KeyPrefix:    "nostr-relay-store-contract/" + time.Now().UTC().Format("20060102150405.000000000"),
	})
	if err != nil {
		t.Fatalf("NewS3Store() error = %v", err)
	}

	runStoreContract(t, store)
}
