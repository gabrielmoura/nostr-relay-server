package conf

import (
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

func TestMarshalStructRedactsStoreS3Credentials(t *testing.T) {
	cfg := &config.Config{Store: config.StoreConfig{S3: config.StoreS3Config{
		AccessKey: "secret-access",
		SecretKey: "secret-key",
	}}}

	data, err := marshalStruct(cfg, FormatYAML)
	if err != nil {
		t.Fatalf("marshalStruct() error = %v", err)
	}
	output := string(data)
	if strings.Contains(output, "secret-access") || strings.Contains(output, "secret-key") {
		t.Fatalf("marshalStruct() leaked credentials: %s", output)
	}
	if !strings.Contains(output, "access_key: \"\"") || !strings.Contains(output, "secret_key: \"\"") {
		t.Fatalf("marshalStruct() did not include redacted credential fields: %s", output)
	}
}
