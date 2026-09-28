package blossom

import (
	"bytes"
	"context"
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
