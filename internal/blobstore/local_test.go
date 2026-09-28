package blobstore

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestLocalStoreContract(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	runStoreContract(t, store)

	ctx := context.Background()
	content := "nostr blossom storage"
	if _, err := store.Put(ctx, testKey, strings.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	fileInfo, err := os.Stat(filepath.Join(store.root, testKey))
	if err != nil {
		t.Fatalf("Stat local file error = %v", err)
	}
	if fileInfo.Mode().Perm() != 0o644 {
		t.Fatalf("stored file permissions = %o, want 644", fileInfo.Mode().Perm())
	}
}

func TestLocalStoreRejectsInvalidInputs(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "invalid key",
			call: func() error {
				_, err := store.Put(context.Background(), "../outside", strings.NewReader("blob"), 4)
				return err
			},
		},
		{
			name: "declared size mismatch",
			call: func() error {
				_, err := store.Put(context.Background(), testKey, strings.NewReader("blob"), 5)
				return err
			},
		},
		{
			name: "range exceeds object",
			call: func() error {
				_, err := store.Put(context.Background(), testKey, strings.NewReader("blob"), 4)
				if err != nil {
					return err
				}
				reader, _, err := store.Get(context.Background(), testKey, ByteRange{Offset: 3, Length: 2})
				if reader != nil {
					_ = reader.Close()
				}
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLocalStoreInvalidRangeReturnsPortableError(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	if _, err := store.Put(context.Background(), testKey, strings.NewReader("blob"), 4); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	reader, _, err := store.Get(context.Background(), testKey, ByteRange{Offset: 3, Length: 2})
	if reader != nil {
		_ = reader.Close()
	}
	if !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("Get() error = %v, want ErrInvalidRange", err)
	}
}

func TestLocalStorePutReadsAtMostDeclaredSizePlusOne(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}

	source := &countingReader{remaining: 100}
	_, err = store.Put(context.Background(), testKey, source, 4)
	if err == nil {
		t.Fatal("Put() error = nil, want oversized body error")
	}
	if source.read != 5 {
		t.Fatalf("Put() consumed %d bytes, want 5", source.read)
	}
}

func TestLocalStorePutStopsWhenContextIsCanceled(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = store.Put(ctx, testKey, &cancelingReader{cancel: cancel}, 2)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Put() error = %v, want context.Canceled", err)
	}
	if _, err := store.Stat(context.Background(), testKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat() error = %v, want ErrNotFound", err)
	}
}

type countingReader struct {
	remaining int
	read      int
}

func (r *countingReader) Read(destination []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := min(len(destination), r.remaining)
	r.remaining -= n
	r.read += n
	for index := range destination[:n] {
		destination[index] = 'x'
	}
	return n, nil
}

type cancelingReader struct {
	cancel context.CancelFunc
	calls  int
}

func (r *cancelingReader) Read(destination []byte) (int, error) {
	r.calls++
	if r.calls == 1 {
		destination[0] = 'x'
		return 1, nil
	}
	r.cancel()
	return 0, io.EOF
}
