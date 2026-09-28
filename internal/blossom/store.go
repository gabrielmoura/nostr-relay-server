package blossom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
)

var (
	blobStoreMu sync.RWMutex
	blobStore   blobstore.Store
)

// SetStore installs the backend used by asynchronous Blossom jobs.
func SetStore(store blobstore.Store) {
	blobStoreMu.Lock()
	defer blobStoreMu.Unlock()

	blobStore = store
}

func currentStore() (blobstore.Store, error) {
	blobStoreMu.RLock()
	defer blobStoreMu.RUnlock()

	if blobStore == nil {
		return nil, fmt.Errorf("blossom blob store is not initialized")
	}

	return blobStore, nil
}

func persistBlob(ctx context.Context, key string, body io.Reader, size int64) error {
	store, err := currentStore()
	if err != nil {
		return err
	}

	info, err := store.Stat(ctx, key)
	if err == nil {
		if info.Size != size {
			return fmt.Errorf("existing blob %q has size %d, expected %d", key, info.Size, size)
		}
		return nil
	}
	if !errors.Is(err, blobstore.ErrNotFound) {
		return fmt.Errorf("inspect existing blob: %w", err)
	}

	_, putErr := store.Put(ctx, key, body, size)
	if putErr == nil {
		return nil
	}

	info, statErr := store.Stat(ctx, key)
	if statErr == nil && info.Size == size {
		return nil
	}
	return fmt.Errorf("persist blob: %w", putErr)
}
