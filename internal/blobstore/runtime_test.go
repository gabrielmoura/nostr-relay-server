package blobstore

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

func TestNewConfiguredStore_Local(t *testing.T) {
	t.Parallel()

	store, err := NewConfiguredStore(context.Background(), config.StoreConfig{Backend: "local"}, t.TempDir())
	if err != nil {
		t.Fatalf("NewConfiguredStore() error = %v", err)
	}
	runStoreContract(t, store)
}

func TestFallbackStore_ReadsLocalOnlyForNotFound(t *testing.T) {
	t.Parallel()

	primary := &stubStore{getErr: ErrNotFound, statErr: ErrNotFound}
	fallback := &stubStore{
		reader: io.NopCloser(strings.NewReader("legacy")),
		info:   ObjectInfo{Key: testKey, Size: 6, Modified: time.Now()},
	}
	store := &fallbackStore{primary: primary, fallback: fallback}

	reader, _, err := store.Get(context.Background(), testKey, ByteRange{Length: -1})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if fallback.getCalls != 1 {
		t.Fatalf("fallback Get() calls = %d, want 1", fallback.getCalls)
	}
	if _, err := store.Stat(context.Background(), testKey); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	primary.getErr = errors.New("S3 unavailable")
	_, _, err = store.Get(context.Background(), testKey, ByteRange{Length: -1})
	if err == nil || fallback.getCalls != 1 {
		t.Fatalf("Get() error = %v, fallback calls = %d; want primary error and no fallback", err, fallback.getCalls)
	}

	primary.putErr = ErrNotFound
	_, err = store.Put(context.Background(), testKey, strings.NewReader("body"), 4)
	if !errors.Is(err, ErrNotFound) || fallback.putCalls != 0 {
		t.Fatalf("Put() error = %v, fallback calls = %d; want primary ErrNotFound and no fallback", err, fallback.putCalls)
	}
}

func TestBlobStoreMetricResult(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{name: "success", want: "success"},
		{name: "not found", err: ErrNotFound, want: "not_found"},
		{name: "invalid range", err: ErrInvalidRange, want: "invalid_range"},
		{name: "canceled", err: context.Canceled, want: "canceled"},
		{name: "other", err: errors.New("network failure"), want: "error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := blobStoreMetricResult(tt.err); got != tt.want {
				t.Fatalf("blobStoreMetricResult() = %q, want %q", got, tt.want)
			}
		})
	}
}

type stubStore struct {
	reader io.ReadCloser
	info   ObjectInfo

	getErr  error
	statErr error
	putErr  error

	getCalls int
	putCalls int
}

func (s *stubStore) Put(context.Context, string, io.Reader, int64) (ObjectInfo, error) {
	s.putCalls++
	return s.info, s.putErr
}

func (s *stubStore) Get(context.Context, string, ByteRange) (io.ReadCloser, ObjectInfo, error) {
	s.getCalls++
	return s.reader, s.info, s.getErr
}

func (s *stubStore) Stat(context.Context, string) (ObjectInfo, error) {
	return s.info, s.statErr
}

func (s *stubStore) Delete(context.Context, string) error { return nil }

func (s *stubStore) List(context.Context, string) ([]ObjectInfo, error) { return []ObjectInfo{}, nil }

var _ Store = (*stubStore)(nil)
