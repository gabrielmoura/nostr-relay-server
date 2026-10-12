package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	infradb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// runRelaySyncIfAvailable probes the independent RelaySync v2 endpoint before
// opening the existing NIP-77 flow. NIP-77 state and message handling are not
// shared with RelaySync; a capability failure only selects the unchanged
// legacy path for peers that do not expose v2.
func runRelaySyncIfAvailable(ctx context.Context, cfg *ConfSync) (*SyncResult, bool, error) {
	if cfg == nil || cfg.Direction == DirectionNone || len(cfg.LocalFilter) != 1 || !isUnrestrictedFilter(cfg.LocalFilter[0]) {
		return nil, false, nil
	}
	baseURL, err := relaySyncHTTPURL(cfg.Remote)
	if err != nil {
		return nil, false, nil
	}
	// Manifest generation is a bounded but potentially expensive database
	// operation. A short fixed timeout aborts valid synchronizations before the
	// first page is produced. The CLI timeout is an explicit operator choice;
	// without it, allow a generous per-request budget while the context still
	// controls cancellation and shutdown.
	httpTimeout := 5 * time.Minute
	if cfg.Timeout > 0 {
		httpTimeout = cfg.Timeout
	}
	client, err := core.NewHTTPClient(core.HTTPClientOptions{
		BaseURL:          baseURL,
		Client:           &http.Client{Timeout: httpTimeout},
		MaxResponseBytes: core.DefaultLimits().MaxBatchBytes,
	})
	if err != nil {
		return nil, false, nil
	}
	caps, err := client.Capabilities(ctx)
	if err != nil {
		if log.Logger != nil {
			log.Logger.Debug("remote does not advertise RelaySync v2; using NIP-77", zap.Error(err), zap.String("remote", cfg.Remote))
		}
		return nil, false, nil
	}
	if !containsVersion(caps.GetProtocolVersions(), 2) {
		return nil, false, nil
	}
	if log.Logger != nil {
		log.Logger.Info("remote advertises RelaySync v2; starting HTTP reconciliation", zap.String("remote", cfg.Remote), zap.String("implementation", caps.GetImplementation()))
	}
	result, err := runRelaySyncHTTP(ctx, client, cfg)
	return result, true, err
}

// RelaySync v2 currently negotiates a named scope, not a NIP-01 filter. Keep
// filtered CLI operations on the legacy path until filter-scoped snapshots are
// part of the negotiated scope contract.
func isUnrestrictedFilter(filter nostr.Filter) bool {
	return len(filter.IDs) == 0 && len(filter.Authors) == 0 && len(filter.Kinds) == 0 &&
		len(filter.Tags) == 0 && filter.Since == nil && filter.Until == nil && filter.Limit == 0 && filter.Search == ""
}

func relaySyncHTTPURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid remote URL %q", raw)
	}
	switch parsed.Scheme {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported remote scheme %q", parsed.Scheme)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func containsVersion(versions []uint32, want uint32) bool {
	for _, version := range versions {
		if version == want {
			return true
		}
	}
	return false
}

func runRelaySyncHTTP(ctx context.Context, client *core.HTTPClient, cfg *ConfSync) (*SyncResult, error) {
	if internaldb.DbQueries == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	scope := core.Scope{Name: "public", Version: 1}
	epoch := config.Cfg.RelaySync.JournalEpoch
	if epoch == 0 {
		epoch = 1
	}
	localProvider, err := infradb.NewRelaySyncSnapshotProvider(internaldb.DbQueries, epoch, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("create local RelaySync snapshot: %w", err)
	}
	localSnapshot, err := localProvider.OpenSnapshot(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("open local RelaySync snapshot: %w", err)
	}
	defer localSnapshot.Close()
	localPartitions := make(map[string]core.PartitionKey, len(localSnapshot.Descriptor().Partitions))
	for _, partition := range localSnapshot.Descriptor().Partitions {
		identity := core.PartitionIdentity(partition)
		localPartitions[string(identity[:])] = partition
	}

	var idempotency [16]byte
	if _, err := rand.Read(idempotency[:]); err != nil {
		return nil, err
	}
	opened, err := client.OpenSession(ctx, &relaysyncv2.OpenSessionRequest{
		ProtocolVersion: 2,
		Compressions:    []relaysyncv2.Compression{relaysyncv2.Compression_COMPRESSION_NONE},
		Scope:           &relaysyncv2.Scope{Name: scope.Name, Version: scope.Version},
		IdempotencyKey:  idempotency[:],
	})
	if err != nil {
		return nil, fmt.Errorf("open RelaySync session: %w", err)
	}
	var sessionID core.SessionID
	if len(opened.GetSessionId()) != len(sessionID) {
		return nil, fmt.Errorf("remote returned invalid RelaySync session id")
	}
	copy(sessionID[:], opened.GetSessionId())
	token := opened.GetResumeToken()
	defer func() {
		if cleanupErr := client.CompleteSession(context.Background(), sessionID, token); cleanupErr != nil {
			if log.Logger != nil {
				log.Logger.Warn("failed to clean up RelaySync session", zap.Error(cleanupErr))
			}
		}
	}()

	var snapshotVersion []byte
	if opened.GetSnapshot() != nil {
		snapshotVersion = opened.GetSnapshot().GetVersion()
	}
	manifestCtx, cancelManifest := context.WithCancel(ctx)
	defer cancelManifest()
	seenRemote := make(map[string]struct{}, len(localPartitions))
	for result := range streamRelaySyncManifest(manifestCtx, client, sessionID, token, snapshotVersion, 10) {
		if result.err != nil {
			return nil, fmt.Errorf("read RelaySync manifest: %w", result.err)
		}
		for _, summary := range result.page.GetPartitions() {
			if summary == nil || summary.GetPartition() == nil || len(summary.GetPartition().GetId()) != 32 {
				return nil, fmt.Errorf("remote returned invalid RelaySync partition")
			}
			rawID := string(summary.GetPartition().GetId())
			seenRemote[rawID] = struct{}{}
			partition, exists := localPartitions[rawID]
			var localItems []core.EventMetadata
			if exists {
				localItems, err = readPartitionMetadata(ctx, localSnapshot, partition)
				if err != nil {
					return nil, err
				}
			}
			var partitionID [32]byte
			copy(partitionID[:], summary.GetPartition().GetId())
			if err := syncRelayPartition(ctx, client, cfg, sessionID, token, localSnapshot, partitionID, localItems, summary); err != nil {
				return nil, err
			}
		}
	}
	if cfg.Direction.DoUp() {
		for rawID, partition := range localPartitions {
			if _, exists := seenRemote[rawID]; exists {
				continue
			}
			localItems, err := readPartitionMetadata(ctx, localSnapshot, partition)
			if err != nil {
				return nil, err
			}
			ids := metadataIDs(localItems)
			var partitionID [32]byte
			copy(partitionID[:], rawID)
			if err := pushSnapshotIDs(ctx, client, sessionID, token, localSnapshot, partitionID, ids); err != nil {
				return nil, err
			}
		}
	}
	return &SyncResult{Filter: filtersToAny(cfg.LocalFilter)}, nil
}

type manifestPageResult struct {
	page *relaysyncv2.Manifest
	err  error
}

func streamRelaySyncManifest(ctx context.Context, client *core.HTTPClient, id core.SessionID, token, snapshot []byte, concurrency int) <-chan manifestPageResult {
	if concurrency <= 0 {
		concurrency = 1
	}
	results := make(chan manifestPageResult)
	var next atomic.Uint64
	var complete atomic.Bool
	var workers sync.WaitGroup
	workers.Add(concurrency)
	for range concurrency {
		go func() {
			defer workers.Done()
			for !complete.Load() {
				offset := next.Add(1) - 1
				continuation := make([]byte, 8)
				binary.BigEndian.PutUint64(continuation, offset)
				page, err := client.ManifestNext(ctx, id, token, &relaysyncv2.ManifestRequest{SessionId: id[:], SnapshotVersion: snapshot, Continuation: continuation, MaxPartitions: 1})
				if err != nil {
					complete.Store(true)
					select {
					case results <- manifestPageResult{err: err}:
					case <-ctx.Done():
					}
					return
				}
				if page.GetComplete() {
					complete.Store(true)
				}
				select {
				case results <- manifestPageResult{page: page}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		workers.Wait()
		close(results)
	}()
	return results
}

func readPartitionMetadata(ctx context.Context, snapshot core.Snapshot, partition core.PartitionKey) ([]core.EventMetadata, error) {
	iterator, err := snapshot.OpenMetadata(ctx, partition, nil)
	if err != nil {
		return nil, fmt.Errorf("open local partition metadata: %w", err)
	}
	defer iterator.Close()
	metadata := make([]core.EventMetadata, 0, 1024)
	for {
		item, ok, err := iterator.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("read local partition metadata: %w", err)
		}
		if !ok {
			return metadata, nil
		}
		metadata = append(metadata, item)
	}
}

func metadataIDs(metadata []core.EventMetadata) []core.EventID {
	ids := make([]core.EventID, len(metadata))
	for i := range metadata {
		ids[i] = metadata[i].ID
	}
	return ids
}

func syncRelayPartition(ctx context.Context, client *core.HTTPClient, cfg *ConfSync, sessionID core.SessionID, token []byte, localSnapshot core.Snapshot, partitionID [32]byte, localItems []core.EventMetadata, remoteSummary *relaysyncv2.PartitionSummary) error {
	localDigest, err := core.DigestMetadata(localItems)
	if err != nil {
		return err
	}
	if string(localDigest[:]) == string(remoteSummary.GetDigest()) {
		return nil
	}
	have, need, err := reconcilePartition(ctx, client, sessionID, token, partitionID, localItems)
	if err != nil {
		return err
	}
	if cfg.Direction.DoDown() {
		if err := pullAndStoreEvents(ctx, client, sessionID, token, need); err != nil {
			return err
		}
	}
	if cfg.Direction.DoUp() {
		if err := pushSnapshotIDs(ctx, client, sessionID, token, localSnapshot, partitionID, have); err != nil {
			return err
		}
	}
	return nil
}

func reconcilePartition(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, partitionID [32]byte, local []core.EventMetadata) ([]core.EventID, []core.EventID, error) {
	engine, err := core.NewNegentropy(local, core.NegentropyOptions{FrameSizeLimit: core.DefaultLimits().MaxFrameBytes})
	if err != nil {
		return nil, nil, err
	}
	frame, err := engine.Initiate()
	if err != nil {
		return nil, nil, err
	}
	var have, need []core.EventID
	for round := uint32(1); round <= uint32(core.DefaultLimits().MaxConcurrentRounds); round++ {
		response, err := client.Reconcile(ctx, id, token, &relaysyncv2.ReconcileRequest{SessionId: id[:], PartitionId: partitionID[:], NegentropyPayload: frame, Round: round})
		if err != nil {
			return nil, nil, fmt.Errorf("reconcile partition: %w", err)
		}
		var diff core.NegentropyDiff
		var done bool
		frame, diff, done, err = engine.Apply(response.GetNegentropyPayload())
		if err != nil {
			return nil, nil, err
		}
		have = append(have, diff.Have...)
		need = append(need, diff.Need...)
		if done || response.GetComplete() {
			return have, need, nil
		}
	}
	return nil, nil, fmt.Errorf("reconcile exceeded maximum rounds")
}

func pullAndStoreEvents(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, ids []core.EventID) error {
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		wireIDs := make([][]byte, end-start)
		for i := range wireIDs {
			wireIDs[i] = append([]byte(nil), ids[start+i][:]...)
		}
		response, err := client.PullEvents(ctx, id, token, &relaysyncv2.PullEventsRequest{SessionId: id[:], OperationId: fmt.Sprintf("pull-%d", start), EventIds: wireIDs, MaxEvents: uint32(len(wireIDs))})
		if err != nil {
			return fmt.Errorf("pull RelaySync events: %w", err)
		}
		events := make([]*nostr.Event, 0, len(response.GetEvents()))
		for _, wire := range response.GetEvents() {
			event, err := core.EventFromProto(wire, core.DefaultLimits())
			if err != nil {
				return err
			}
			events = append(events, nostrEventFromCore(event))
		}
		if len(events) > 0 {
			if _, err := internaldb.DbQueries.InsertEventBatch(ctx, events); err != nil {
				return fmt.Errorf("store pulled events: %w", err)
			}
		}
	}
	return nil
}

func pushSnapshotIDs(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, snapshot core.Snapshot, partitionID [32]byte, ids []core.EventID) error {
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		iterator, err := snapshot.GetEvents(ctx, ids[start:end])
		if err != nil {
			return fmt.Errorf("read local RelaySync events: %w", err)
		}
		wireEvents := make([]*relaysyncv2.Event, 0, end-start)
		for {
			item, ok, nextErr := iterator.Next(ctx)
			if nextErr != nil {
				_ = iterator.Close()
				return fmt.Errorf("iterate local RelaySync events: %w", nextErr)
			}
			if !ok {
				break
			}
			if item.Event != nil {
				wireEvents = append(wireEvents, core.EventToProto(*item.Event))
			}
		}
		if err := iterator.Close(); err != nil {
			return fmt.Errorf("close local RelaySync event iterator: %w", err)
		}
		if len(wireEvents) == 0 {
			continue
		}
		// Operation IDs are session-scoped on the server. Include the complete
		// partition identity so separate batches cannot be treated as a retry of
		// another partition (which would correctly result in HTTP 409).
		operationID := fmt.Sprintf("push-%x-%d", partitionID, start)
		response, err := client.PushEvents(ctx, id, token, &relaysyncv2.PushEventsRequest{SessionId: id[:], OperationId: operationID, Events: wireEvents, IdempotencyKey: []byte(operationID)})
		if err != nil {
			return fmt.Errorf("push RelaySync events: %w", err)
		}
		if err := handleRelaySyncPushReceipts(response.GetReceipts(), wireEvents); err != nil {
			return err
		}
	}
	return nil
}

// handleRelaySyncPushReceipts makes permanent remote policy or validation
// decisions visible without aborting an otherwise valid finite sync cycle.
// A temporary decision remains fatal so it can be retried instead of being
// silently treated as convergence.
func handleRelaySyncPushReceipts(receipts []*relaysyncv2.EventReceipt, events []*relaysyncv2.Event) error {
	if len(receipts) != len(events) {
		return fmt.Errorf("push RelaySync events: received %d receipts for %d events", len(receipts), len(events))
	}
	for index, receipt := range receipts {
		if receipt == nil {
			return fmt.Errorf("push RelaySync events: received empty receipt")
		}
		if !bytes.Equal(receipt.GetId(), events[index].GetId()) {
			return fmt.Errorf("push RelaySync events: receipt id does not match sent event")
		}
		switch receipt.GetStatus() {
		case relaysyncv2.WriteStatus_WRITE_ACCEPTED, relaysyncv2.WriteStatus_WRITE_ALREADY_EXISTS:
			continue
		case relaysyncv2.WriteStatus_WRITE_REJECTED_PERMANENT:
			if log.Logger != nil {
				log.Logger.Warn("RelaySync event permanently rejected by remote", zap.String("event_id", hex.EncodeToString(receipt.GetId())), zap.String("code", receipt.GetCode()), zap.String("reason", receipt.GetMessage()))
			}
			continue
		case relaysyncv2.WriteStatus_WRITE_FAILED_TEMPORARY:
			return fmt.Errorf("push RelaySync events: remote temporary failure (%s)", receipt.GetCode())
		default:
			return fmt.Errorf("push RelaySync events: remote returned invalid receipt status (%s)", receipt.GetStatus().String())
		}
	}
	return nil
}

func nostrEventFromCore(event core.Event) *nostr.Event {
	tags := make(nostr.Tags, len(event.Tags))
	for i := range event.Tags {
		tags[i] = append(nostr.Tag(nil), event.Tags[i]...)
	}
	return &nostr.Event{ID: event.ID.String(), PubKey: event.PublicKey.String(), Sig: event.Signature.String(), CreatedAt: nostr.Timestamp(event.CreatedAt), Kind: int(event.Kind), Tags: tags, Content: event.Content}
}
