package magic

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
)

type lookupResult struct {
	fileType *FileType
	found    bool
}

// LookupConfig is a struct that contains configuration details to modify the default Lookup behavior
type LookupConfig struct {
	ConcurrencyEnabled bool
	WorkerCount        int
}

// ErrUnknown infers the file type cannot be determined by the provided magic bytes
var ErrUnknown = fmt.Errorf("unknown file type")

// Lookup looks up the file type based on the provided magic bytes.
func Lookup(bytes []byte) (*FileType, error) {
	return lookup(bytes, true, -1)
}

// LookupWithConfig looks up the file type based on the provided magic bytes, and a given configuration.
func LookupWithConfig(bytes []byte, config LookupConfig) (*FileType, error) {
	return lookup(bytes, config.ConcurrencyEnabled, config.WorkerCount)
}

// LookupSync lookups up the file type based on the provided magic bytes without spawning any additional goroutines.
func LookupSync(bytes []byte) (*FileType, error) {
	return lookup(bytes, false, 0)
}

func lookup(bytes []byte, concurrent bool, workers int) (*FileType, error) {
	// Sort types by offset in descending order
	sortedTypes := make([]FileType, len(Types))
	copy(sortedTypes, Types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Offset > sortedTypes[j].Offset
	})

	if !concurrent || workers == 0 {
		for _, t := range sortedTypes {
			ft := t.check(bytes, t.Offset)
			if ft != nil {
				return ft, nil
			}
		}
		return nil, ErrUnknown
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workers > -1 && workers < workerCount {
		workerCount = workers
	}

	workChan := make(chan FileType, len(sortedTypes)) // Buffer igual ao número de tipos
	for _, t := range sortedTypes {
		workChan <- t
	}
	close(workChan)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := lookupResult{nil, false}
	var resultMutex sync.Mutex

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(ctx, workChan, bytes, &result, &resultMutex, &wg, cancel)
	}

	wg.Wait()

	if result.found {
		return result.fileType, nil
	}

	return nil, ErrUnknown
}

func worker(ctx context.Context, workChan chan FileType, bytes []byte, result *lookupResult, resultMutex *sync.Mutex, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case fileType, ok := <-workChan:
			if !ok {
				return // Channel closed
			}
			ft := fileType.check(bytes, fileType.Offset)
			if ft != nil {
				resultMutex.Lock()
				if !result.found {
					result.fileType = ft
					result.found = true
					cancel() // Signal to other workers to stop
				}
				resultMutex.Unlock()
				return // Stop worker when a type was found
			}

		}
	}
}
