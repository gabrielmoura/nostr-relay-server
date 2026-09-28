package blobstore

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func runStoreContract(t *testing.T, store Store) {
	t.Helper()

	ctx := context.Background()
	content := "nostr blossom storage"
	object, err := store.Put(ctx, testKey, strings.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if object.Key != testKey || object.Size != int64(len(content)) || object.Modified.IsZero() {
		t.Fatalf("Put() object = %#v", object)
	}

	stat, err := store.Stat(ctx, testKey)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if stat.Key != testKey || stat.Size != int64(len(content)) {
		t.Fatalf("Stat() object = %#v", stat)
	}

	reader, gotInfo, err := store.Get(ctx, testKey, ByteRange{Offset: 6, Length: 7})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	got, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if string(got) != "blossom" || gotInfo != stat {
		t.Fatalf("Get() = %q, %#v; want %q, %#v", got, gotInfo, "blossom", stat)
	}

	reader, _, err = store.Get(ctx, testKey, ByteRange{Offset: 14, Length: -1})
	if err != nil {
		t.Fatalf("Get() remainder error = %v", err)
	}
	got, readErr = io.ReadAll(reader)
	closeErr = reader.Close()
	if readErr != nil {
		t.Fatalf("ReadAll() remainder error = %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("Close() remainder error = %v", closeErr)
	}
	if string(got) != "storage" {
		t.Fatalf("Get() remainder = %q, want %q", got, "storage")
	}

	objects, err := store.List(ctx, testKey[:8])
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 || objects[0].Key != testKey {
		t.Fatalf("List() = %#v", objects)
	}

	if err := store.Delete(ctx, testKey); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = store.Stat(ctx, testKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat() error = %v, want ErrNotFound", err)
	}
}
