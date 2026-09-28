package blossommigrate

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
)

// Result summarizes a migration without exposing individual object keys.
type Result struct {
	Scanned      int
	Migrated     int
	Skipped      int
	WouldMigrate int
}

// Run copies every local blob absent from the target. Existing target objects
// are skipped only when their size matches the source. It never deletes source
// objects or replaces conflicting target objects.
func Run(ctx context.Context, raw Options, source, target blobstore.Store) (Result, error) {
	options, err := BuildOptions(raw)
	if err != nil {
		return Result{}, err
	}
	if source == nil {
		return Result{}, errors.New("migration source store cannot be nil")
	}
	if target == nil {
		return Result{}, errors.New("migration target store cannot be nil")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	objects, err := source.List(ctx, "")
	if err != nil {
		return Result{}, fmt.Errorf("list local Blossom blobs: %w", err)
	}

	result := Result{Scanned: len(objects)}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan blobstore.ObjectInfo)
	var (
		workers  sync.WaitGroup
		resultMu sync.Mutex
		firstErr error
		errOnce  sync.Once
	)
	recordError := func(err error) {
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	for range options.Workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for object := range jobs {
				if err := migrateObject(runCtx, source, target, object, options.DryRun, &result, &resultMu); err != nil {
					recordError(err)
					return
				}
			}
		}()
	}

	for _, object := range objects {
		select {
		case <-runCtx.Done():
			break
		case jobs <- object:
		}
		if runCtx.Err() != nil {
			break
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return result, firstErr
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func migrateObject(ctx context.Context, source, target blobstore.Store, sourceInfo blobstore.ObjectInfo, dryRun bool, result *Result, resultMu *sync.Mutex) error {
	targetInfo, err := target.Stat(ctx, sourceInfo.Key)
	if err == nil {
		if targetInfo.Size != sourceInfo.Size {
			return fmt.Errorf("target blob %q has size %d, expected %d", sourceInfo.Key, targetInfo.Size, sourceInfo.Size)
		}
		resultMu.Lock()
		result.Skipped++
		resultMu.Unlock()
		return nil
	}
	if !errors.Is(err, blobstore.ErrNotFound) {
		return fmt.Errorf("stat target blob %q: %w", sourceInfo.Key, err)
	}
	if dryRun {
		resultMu.Lock()
		result.WouldMigrate++
		resultMu.Unlock()
		return nil
	}

	reader, readInfo, err := source.Get(ctx, sourceInfo.Key, blobstore.ByteRange{Length: -1})
	if err != nil {
		return fmt.Errorf("read local blob %q: %w", sourceInfo.Key, err)
	}
	defer reader.Close()
	if readInfo.Size != sourceInfo.Size {
		return fmt.Errorf("local blob %q changed size during migration: listed %d, read %d", sourceInfo.Key, sourceInfo.Size, readInfo.Size)
	}

	writtenInfo, err := target.Put(ctx, sourceInfo.Key, reader, sourceInfo.Size)
	if err != nil {
		return fmt.Errorf("copy blob %q to S3: %w", sourceInfo.Key, err)
	}
	if writtenInfo.Size != sourceInfo.Size {
		return fmt.Errorf("target blob %q has size %d after copy, expected %d", sourceInfo.Key, writtenInfo.Size, sourceInfo.Size)
	}
	resultMu.Lock()
	result.Migrated++
	resultMu.Unlock()
	return nil
}
