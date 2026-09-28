package blossom

import (
	"bytes"
	"context"
	"database/sql"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
)

func TestBuildNIP94Tags(t *testing.T) {
	t.Parallel()

	width := int32(640)
	height := int32(480)
	config.Cfg = &config.Config{}
	config.Cfg.Store.MediaPath = "https://cdn.example.com/blob"
	object := dbmodel.Object{
		Hash:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MimeType:  "image/png",
		Size:      12345,
		CreatedAt: time.Now().UTC(),
	}

	tags := buildNIP94Tags(
		object,
		&width,
		&height,
		nil,
		nil,
		"blur-value",
		"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
		"",
		[]string{"https://mirror.example.com/blob/1", "https://mirror.example.com/blob/1", "https://mirror.example.com/blob/2"},
	)

	want := [][]string{
		{"url", directURL(object.Hash)},
		{"m", "image/png"},
		{"x", object.Hash},
		{"size", "12345"},
		{"service", "nip96"},
		{"dim", "640x480"},
		{"blurhash", "blur-value"},
		{"thumb", directURL("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"), "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"},
		{"image", directURL("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"), "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"},
		{"ox", object.Hash},
		{"fallback", "https://mirror.example.com/blob/1"},
		{"fallback", "https://mirror.example.com/blob/2"},
	}

	if len(tags) != len(want) {
		t.Fatalf("len(tags) = %d, want %d", len(tags), len(want))
	}
	for i := range want {
		if len(tags[i]) != len(want[i]) {
			t.Fatalf("len(tags[%d]) = %d, want %d", i, len(tags[i]), len(want[i]))
		}
		for j := range want[i] {
			if tags[i][j] != want[i][j] {
				t.Fatalf("tags[%d][%d] = %q, want %q", i, j, tags[i][j], want[i][j])
			}
		}
	}
}

func TestMaterializeBlob(t *testing.T) {
	store, err := blobstore.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	payload := []byte("media source")
	if _, err := store.Put(context.Background(), key, bytes.NewReader(payload), int64(len(payload))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	path, err := materializeBlob(context.Background(), key)
	if err != nil {
		t.Fatalf("materializeBlob() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("materialized payload = %q, want %q", got, payload)
	}

}

func TestPersistBlobIsIdempotent(t *testing.T) {
	store, err := blobstore.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	key := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	payload := []byte("derivative")

	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })

	for range 2 {
		if err := persistBlob(context.Background(), key, bytes.NewReader(payload), int64(len(payload))); err != nil {
			t.Fatalf("persistBlob() error = %v", err)
		}
	}
}

func TestValidateMirrorURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "public HTTPS address", url: "https://8.8.8.8/blob", want: true},
		{name: "loopback IPv4", url: "http://127.0.0.1/blob"},
		{name: "private IPv4", url: "http://10.0.0.1/blob"},
		{name: "link local IPv4", url: "http://169.254.169.254/latest/meta-data"},
		{name: "shared IPv4", url: "http://100.64.0.1/blob"},
		{name: "loopback IPv6", url: "http://[::1]/blob"},
		{name: "private IPv6", url: "http://[fd00::1]/blob"},
		{name: "credentials", url: "https://user:password@example.com/blob"},
		{name: "unsupported scheme", url: "file:///etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("url.Parse() error = %v", err)
			}
			err = validateMirrorURL(parsed)
			if (err == nil) != tt.want {
				t.Fatalf("validateMirrorURL(%q) error = %v, want success = %t", tt.url, err, tt.want)
			}
		})
	}
}

func TestIsPublicMirrorIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "8.8.8.8", want: true},
		{name: "private IPv4", ip: "192.168.1.1"},
		{name: "multicast IPv4", ip: "224.0.0.1"},
		{name: "public IPv6", ip: "2001:4860:4860::8888", want: true},
		{name: "private IPv6", ip: "fc00::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPublicMirrorIP(net.ParseIP(tt.ip)); got != tt.want {
				t.Fatalf("isPublicMirrorIP(%q) = %t, want %t", tt.ip, got, tt.want)
			}
		})
	}
}

func TestMirrorDownloadLimit(t *testing.T) {
	t.Parallel()

	quota := int64(100)
	policy := dbmodel.BlossomServerPolicy{DefaultStorageQuotaBytes: sql.NullInt64{Int64: quota, Valid: true}}
	if got := mirrorDownloadLimit(policy, nil, 40); got != 60 {
		t.Fatalf("mirrorDownloadLimit() = %d, want 60", got)
	}
	if got := mirrorDownloadLimit(policy, nil, 100); got != 0 {
		t.Fatalf("mirrorDownloadLimit() at quota = %d, want 0", got)
	}
	if got := mirrorDownloadLimit(dbmodel.BlossomServerPolicy{}, nil, 40); got != -1 {
		t.Fatalf("mirrorDownloadLimit() without quota = %d, want -1", got)
	}
}
