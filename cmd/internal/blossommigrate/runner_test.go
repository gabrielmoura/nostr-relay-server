package blossommigrate

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
)

const key = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestRunCopiesMissingObjectsAndNeverDeletesSource(t *testing.T) {
	source := newMemoryStore(map[string]string{key: "blossom"})
	target := newMemoryStore(nil)

	result, err := Run(context.Background(), Options{From: "local", To: "s3", Workers: 2}, source, target)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result != (Result{Scanned: 1, Migrated: 1}) {
		t.Fatalf("Run() result = %#v", result)
	}
	if got := target.content(key); got != "blossom" {
		t.Fatalf("target content = %q, want blossom", got)
	}
	if source.deleteCalls != 0 || source.content(key) != "blossom" {
		t.Fatalf("source was modified: deletes=%d content=%q", source.deleteCalls, source.content(key))
	}
}

func TestRunSkipsMatchingTargetAndRejectsSizeConflict(t *testing.T) {
	t.Run("matching target is skipped", func(t *testing.T) {
		source := newMemoryStore(map[string]string{key: "blossom"})
		target := newMemoryStore(map[string]string{key: "blossom"})

		result, err := Run(context.Background(), Options{From: "local", To: "s3", Workers: 1}, source, target)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if result != (Result{Scanned: 1, Skipped: 1}) || target.putCalls != 0 {
			t.Fatalf("Run() result = %#v, puts=%d", result, target.putCalls)
		}
	})

	t.Run("size conflict is not overwritten", func(t *testing.T) {
		source := newMemoryStore(map[string]string{key: "blossom"})
		target := newMemoryStore(map[string]string{key: "other"})

		_, err := Run(context.Background(), Options{From: "local", To: "s3", Workers: 1}, source, target)
		if err == nil || !strings.Contains(err.Error(), "expected 7") {
			t.Fatalf("Run() error = %v, want size conflict", err)
		}
		if target.putCalls != 0 || target.content(key) != "other" {
			t.Fatalf("target conflict was modified: puts=%d content=%q", target.putCalls, target.content(key))
		}
	})
}

func TestRunDryRunDoesNotReadOrWrite(t *testing.T) {
	source := newMemoryStore(map[string]string{key: "blossom"})
	target := newMemoryStore(nil)

	result, err := Run(context.Background(), Options{From: "local", To: "s3", DryRun: true, Workers: 1}, source, target)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result != (Result{Scanned: 1, WouldMigrate: 1}) {
		t.Fatalf("Run() result = %#v", result)
	}
	if source.getCalls != 0 || target.putCalls != 0 {
		t.Fatalf("dry run used data path: gets=%d puts=%d", source.getCalls, target.putCalls)
	}
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		wantErr bool
	}{
		{name: "valid", options: Options{From: " LOCAL ", To: "S3", Workers: 1}},
		{name: "invalid source", options: Options{From: "s3", To: "s3", Workers: 1}, wantErr: true},
		{name: "invalid target", options: Options{From: "local", To: "local", Workers: 1}, wantErr: true},
		{name: "invalid workers", options: Options{From: "local", To: "s3"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildOptions() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

type memoryStore struct {
	mu          sync.Mutex
	objects     map[string]string
	getCalls    int
	putCalls    int
	deleteCalls int
}

func newMemoryStore(objects map[string]string) *memoryStore {
	copy := make(map[string]string, len(objects))
	for objectKey, content := range objects {
		copy[objectKey] = content
	}
	return &memoryStore{objects: copy}
}

func (s *memoryStore) Put(_ context.Context, objectKey string, body io.Reader, size int64) (blobstore.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.putCalls++
	content, err := io.ReadAll(body)
	if err != nil {
		return blobstore.ObjectInfo{}, err
	}
	if int64(len(content)) != size {
		return blobstore.ObjectInfo{}, errors.New("size mismatch")
	}
	s.objects[objectKey] = string(content)
	return s.info(objectKey), nil
}

func (s *memoryStore) Get(_ context.Context, objectKey string, _ blobstore.ByteRange) (io.ReadCloser, blobstore.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	if _, ok := s.objects[objectKey]; !ok {
		return nil, blobstore.ObjectInfo{}, blobstore.ErrNotFound
	}
	return io.NopCloser(strings.NewReader(s.objects[objectKey])), s.info(objectKey), nil
}

func (s *memoryStore) Stat(_ context.Context, objectKey string) (blobstore.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.objects[objectKey]; !ok {
		return blobstore.ObjectInfo{}, blobstore.ErrNotFound
	}
	return s.info(objectKey), nil
}

func (s *memoryStore) Delete(_ context.Context, objectKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalls++
	delete(s.objects, objectKey)
	return nil
}

func (s *memoryStore) List(_ context.Context, _ string) ([]blobstore.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	objects := make([]blobstore.ObjectInfo, 0, len(s.objects))
	for objectKey := range s.objects {
		objects = append(objects, s.info(objectKey))
	}
	return objects, nil
}

func (s *memoryStore) content(objectKey string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.objects[objectKey]
}

func (s *memoryStore) info(objectKey string) blobstore.ObjectInfo {
	return blobstore.ObjectInfo{Key: objectKey, Size: int64(len(s.objects[objectKey])), Modified: time.Unix(0, 0).UTC()}
}

var _ blobstore.Store = (*memoryStore)(nil)
