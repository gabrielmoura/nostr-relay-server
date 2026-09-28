package blobstore

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
)

// instrumentedStore records backend-neutral operation metrics. The backend
// label is configured at startup and never contains endpoint or bucket data.
type instrumentedStore struct {
	store   Store
	backend string
}

func newInstrumentedStore(store Store, backend string) Store {
	return &instrumentedStore{store: store, backend: backend}
}

func (s *instrumentedStore) Put(ctx context.Context, key string, body io.Reader, size int64) (ObjectInfo, error) {
	startedAt := time.Now()
	info, err := s.store.Put(ctx, key, body, size)
	s.observe("put", startedAt, err)
	return info, err
}

func (s *instrumentedStore) Get(ctx context.Context, key string, byteRange ByteRange) (io.ReadCloser, ObjectInfo, error) {
	startedAt := time.Now()
	reader, info, err := s.store.Get(ctx, key, byteRange)
	s.observe("get", startedAt, err)
	return reader, info, err
}

func (s *instrumentedStore) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	startedAt := time.Now()
	info, err := s.store.Stat(ctx, key)
	s.observe("stat", startedAt, err)
	return info, err
}

func (s *instrumentedStore) Delete(ctx context.Context, key string) error {
	startedAt := time.Now()
	err := s.store.Delete(ctx, key)
	s.observe("delete", startedAt, err)
	return err
}

func (s *instrumentedStore) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	startedAt := time.Now()
	objects, err := s.store.List(ctx, prefix)
	s.observe("list", startedAt, err)
	return objects, err
}

func (s *instrumentedStore) observe(operation string, startedAt time.Time, err error) {
	result := blobStoreMetricResult(err)
	metrics.NostrBlobStoreOperationsTotal.WithLabelValues(s.backend, operation, result).Inc()
	metrics.NostrBlobStoreOperationDurationSeconds.WithLabelValues(s.backend, operation, result).Observe(time.Since(startedAt).Seconds())
}

func blobStoreMetricResult(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrInvalidRange):
		return "invalid_range"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled"
	default:
		return "error"
	}
}

var _ Store = (*instrumentedStore)(nil)
