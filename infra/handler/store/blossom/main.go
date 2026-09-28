package blossom

import (
	"fmt"
	"slices"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
)

var (
	blobPath   = "files"
	bufferSize = 2 << 9
	storeMu    sync.RWMutex
	store      blobstore.Store
)

// SetStore installs the process-wide blob backend during server bootstrap.
func SetStore(value blobstore.Store) {
	storeMu.Lock()
	defer storeMu.Unlock()
	store = value
}

func currentStore() (blobstore.Store, error) {
	storeMu.RLock()
	defer storeMu.RUnlock()
	if store == nil {
		return nil, fmt.Errorf("blossom blob store is not initialized")
	}
	return store, nil
}

func acceptMethods(method string, methods []string) bool {
	return slices.Contains(methods, method)
}

func acceptMimeType(mimeType string, acceptedMimetypes []string) bool {
	return slices.Contains(acceptedMimetypes, mimeType)
}
