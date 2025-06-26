package nip77

import (
	"context"
	"fmt"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy/storage/vector"
)

type direction struct {
	label  string
	items  chan string
	source nostr.RelayStore
	target nostr.RelayStore
}

// NegentropySync syncs the local store with the remote store using the NIP-77 Negentropy protocol.
//
// Parameters:
//   - ctx: context for cancellation
//   - store: local RelayStore implementation
//   - url: remote relay URL
//   - filter: nostr.Filter to use for selecting events
//   - d: direction ("up", "down", or "both")
func NegentropySync(ctx context.Context, store nostr.RelayStore, url string, filter nostr.Filter, d string) error {
	id := "go-nostr-tmp"

	neg := prepareLocalVector(store, ctx, filter)
	result := make(chan error)

	r, err := setupRelayConnection(ctx, url, id, neg, result)
	if err != nil {
		return err
	}
	defer closeRelay(r, id)

	if err := sendOpenEnvelope(r, id, filter, neg.Start()); err != nil {
		return err
	}

	directions := defineDirections(neg, store, r)
	startDirectionWorkers(ctx, directions, d, result)

	return <-result
}

// prepareLocalVector queries the local store and prepares the sealed vector.
func prepareLocalVector(store nostr.RelayStore, ctx context.Context, filter nostr.Filter) *negentropy.Negentropy {
	data, err := store.QuerySync(ctx, filter)
	if err != nil {
		panic(fmt.Errorf("failed to query local store: %w", err))
	}

	vec := vector.New()
	for _, evt := range data {
		vec.Insert(evt.CreatedAt, evt.ID)
	}
	vec.Seal()

	return negentropy.New(vec, 1024*1024)
}

// setupRelayConnection connects to the relay and sets up the Negentropy handler.
func setupRelayConnection(ctx context.Context, url, id string, neg *negentropy.Negentropy, result chan error) (*nostr.Relay, error) {
	var r *nostr.Relay
	var err error
	r, err = nostr.RelayConnect(ctx, url, nostr.WithCustomHandler(func(data []byte) {
		envelope := ParseNegMessage(data)
		if envelope == nil {
			return
		}
		switch env := envelope.(type) {
		case *OpenEnvelope, *CloseEnvelope:
			result <- fmt.Errorf("unexpected %s received from relay", env.Label())
		case *ErrorEnvelope:
			result <- fmt.Errorf("relay returned a %s: %s", env.Label(), env.Reason)
		case *MessageEnvelope:
			msg, err := neg.Reconcile(env.Message)
			if err != nil {
				result <- fmt.Errorf("failed to reconcile: %w", err)
				return
			}
			if msg != "" {
				msgb, _ := MessageEnvelope{id, msg}.MarshalJSON()
				r.Write(msgb)
			}
		}
	}))
	return r, err
}

// sendOpenEnvelope sends the initial open envelope with the negentropy message.
func sendOpenEnvelope(r *nostr.Relay, id string, filter nostr.Filter, msg string) error {
	open, _ := OpenEnvelope{id, filter, msg}.MarshalJSON()
	if err := <-r.Write(open); err != nil {
		return fmt.Errorf("failed to write open envelope: %w", err)
	}
	return nil
}

// closeRelay sends the close envelope to terminate the sync session.
func closeRelay(r *nostr.Relay, id string) {
	clse, _ := CloseEnvelope{id}.MarshalJSON()
	r.Write(clse)
}

// defineDirections returns the available directions and their handlers.
func defineDirections(neg *negentropy.Negentropy, store, relay nostr.RelayStore) map[string][]direction {
	return map[string][]direction{
		"up":   {{"up", neg.Haves, store, relay}},
		"down": {{"down", neg.HaveNots, relay, store}},
		"both": {{"up", neg.Haves, store, relay}, {"down", neg.HaveNots, relay, store}},
	}
}

// startDirectionWorkers starts goroutines for each selected sync direction.
func startDirectionWorkers(ctx context.Context, directions map[string][]direction, d string, result chan error) {
	var wg sync.WaitGroup
	pool := newidlistpool(50)

	for _, dir := range selectDir(directions, d) {
		wg.Add(1)
		go func(dir direction) {
			defer wg.Done()
			performSync(ctx, dir, &wg, pool, result)
		}(dir)
	}

	go func() {
		wg.Wait()
		result <- nil
	}()
}

// performSync processes events in a specific direction using batched IDs.
func performSync(ctx context.Context, dir direction, wg *sync.WaitGroup, pool *idlistpool, result chan error) {
	seen := make(map[string]struct{})
	ids := pool.grab()

	for item := range dir.items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		ids = append(ids, item)

		if len(ids) == 50 {
			wg.Add(1)
			go fetchAndPublish(ctx, dir, ids, wg, pool, result)
			ids = pool.grab()
		}
	}
	wg.Add(1)
	fetchAndPublish(ctx, dir, ids, wg, pool, result)
}

// fetchAndPublish fetches events by ID and republishes to the target.
func fetchAndPublish(ctx context.Context, dir direction, ids []string, wg *sync.WaitGroup, pool *idlistpool, result chan error) {
	defer wg.Done()
	defer pool.giveback(ids)

	if len(ids) == 0 {
		return
	}

	evtch, err := dir.source.QueryEvents(ctx, nostr.Filter{IDs: ids})
	if err != nil {
		result <- fmt.Errorf("error querying source on %s: %w", dir.label, err)
		return
	}

	for evt := range evtch {
		dir.target.Publish(ctx, *evt)
	}
}

// selectDir returns the directions to sync based on the user input.
//
// If input is invalid or empty, defaults to "both".
func selectDir[T any](directions map[string][]T, d string) []T {
	switch d {
	case "up":
		return directions["up"]
	case "down":
		return directions["down"]
	default:
		return directions["both"]
	}
}
