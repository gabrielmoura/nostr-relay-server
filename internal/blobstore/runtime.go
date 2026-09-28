package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

// NewConfiguredStore selects the configured blob backend. The local root is
// always created for the local backend and only created as a read fallback for
// S3 when explicitly enabled.
func NewConfiguredStore(ctx context.Context, cfg config.StoreConfig, localRoot string) (Store, error) {
	switch cfg.NormalizedBackend() {
	case "", "local":
		store, err := NewLocalStore(localRoot)
		if err != nil {
			return nil, fmt.Errorf("create local blob store: %w", err)
		}
		return newInstrumentedStore(store, "local"), nil
	case "s3":
		primary, err := NewS3Store(ctx, cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("create S3 blob store: %w", err)
		}
		if !cfg.S3.FallbackLocal {
			return newInstrumentedStore(primary, "s3"), nil
		}

		fallback, err := NewLocalStore(localRoot)
		if err != nil {
			return nil, fmt.Errorf("create local blob fallback: %w", err)
		}
		return newInstrumentedStore(&fallbackStore{primary: primary, fallback: fallback}, "s3"), nil
	default:
		return nil, fmt.Errorf("unsupported blob backend %q", cfg.Backend)
	}
}

// fallbackStore serves legacy local objects only when the primary backend
// explicitly reports that an object does not exist. Writes remain primary-only
// so an S3 outage cannot silently create divergent object sets.
type fallbackStore struct {
	primary  Store
	fallback Store
}

func (s *fallbackStore) Put(ctx context.Context, key string, body io.Reader, size int64) (ObjectInfo, error) {
	return s.primary.Put(ctx, key, body, size)
}

func (s *fallbackStore) Get(ctx context.Context, key string, byteRange ByteRange) (io.ReadCloser, ObjectInfo, error) {
	reader, info, err := s.primary.Get(ctx, key, byteRange)
	if !errors.Is(err, ErrNotFound) {
		return reader, info, err
	}
	return s.fallback.Get(ctx, key, byteRange)
}

func (s *fallbackStore) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	info, err := s.primary.Stat(ctx, key)
	if !errors.Is(err, ErrNotFound) {
		return info, err
	}
	return s.fallback.Stat(ctx, key)
}

func (s *fallbackStore) Delete(ctx context.Context, key string) error {
	return s.primary.Delete(ctx, key)
}

func (s *fallbackStore) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	return s.primary.List(ctx, prefix)
}

var _ Store = (*fallbackStore)(nil)
