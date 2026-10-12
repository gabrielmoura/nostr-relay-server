package sync

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// runRelaySyncIfAvailable probes the remote endpoint before opening the
// legacy NIP-77 WebSocket. A capability request failure means the peer is not
// exposing RelaySync v2 and is intentionally treated as a legacy peer.
func runRelaySyncIfAvailable(ctx context.Context, cfg *ConfSync) (*SyncResult, bool, error) {
	if cfg == nil || cfg.Direction == DirectionNone {
		return nil, false, nil
	}
	baseURL, err := relaySyncHTTPURL(cfg.Remote)
	if err != nil {
		return nil, false, nil
	}
	httpTimeout := 15 * time.Second
	if cfg.Timeout > 0 && cfg.Timeout < httpTimeout {
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
	if db.DbQueries() == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	localEvents, err := db.DbQueries().QueryEvents(ctx, cfg.LocalFilter[0])
	if err != nil {
		return nil, fmt.Errorf("query local events: %w", err)
	}
	localByID := make(map[string]*nostr.Event, len(localEvents))
	localMetadata := make(map[core.PartitionKey][]core.EventMetadata)
	for _, event := range localEvents {
		if event == nil {
			continue
		}
		id, err := core.ParseEventID(event.ID)
		if err != nil {
			return nil, fmt.Errorf("local event id %q: %w", event.ID, err)
		}
		localByID[id.String()] = event
		createdAt := int64(event.CreatedAt)
		partition := core.PartitionFor(core.EventMetadata{ID: id, CreatedAt: createdAt}, 8)
		localMetadata[partition] = append(localMetadata[partition], core.EventMetadata{ID: id, CreatedAt: createdAt})
	}
	for partition := range localMetadata {
		sort.Slice(localMetadata[partition], func(i, j int) bool {
			left, right := localMetadata[partition][i], localMetadata[partition][j]
			if left.CreatedAt != right.CreatedAt {
				return left.CreatedAt < right.CreatedAt
			}
			return left.ID.String() < right.ID.String()
		})
	}

	var idempotency [16]byte
	if _, err := rand.Read(idempotency[:]); err != nil {
		return nil, err
	}
	opened, err := client.OpenSession(ctx, &relaysyncv2.OpenSessionRequest{
		ProtocolVersion: 2,
		Compressions:    []relaysyncv2.Compression{relaysyncv2.Compression_COMPRESSION_NONE},
		Scope:           &relaysyncv2.Scope{Name: "public", Version: 1},
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
	remotePartitions, err := fetchRelaySyncManifest(ctx, client, sessionID, token, snapshotVersion)
	if err != nil {
		return nil, err
	}
	remoteByID := make(map[string]*relaysyncv2.PartitionSummary, len(remotePartitions))
	for _, summary := range remotePartitions {
		if summary != nil && summary.GetPartition() != nil {
			remoteByID[string(summary.GetPartition().GetId())] = summary
		}
	}

	partitionIDs := make(map[string]struct{}, len(remoteByID)+len(localMetadata))
	for id := range remoteByID {
		partitionIDs[id] = struct{}{}
	}
	for partition := range localMetadata {
		id := core.PartitionIdentity(partition)
		partitionIDs[string(id[:])] = struct{}{}
	}
	for rawID := range partitionIDs {
		var partitionID [32]byte
		copy(partitionID[:], rawID)
		_, localItems := partitionForID(localMetadata, partitionID)
		remoteSummary := remoteByID[rawID]
		if remoteSummary == nil {
			if cfg.Direction.DoUp() {
				if err := pushLocalEvents(ctx, client, sessionID, token, localItems, localByID); err != nil {
					return nil, err
				}
			}
			continue
		}
		localDigest, err := core.DigestMetadata(localItems)
		if err != nil {
			return nil, err
		}
		if string(localDigest[:]) == string(remoteSummary.GetDigest()) {
			continue
		}
		have, need, err := reconcilePartition(ctx, client, sessionID, token, partitionID, localItems)
		if err != nil {
			return nil, err
		}
		if cfg.Direction.DoDown() {
			if err := pullAndStoreEvents(ctx, client, sessionID, token, need); err != nil {
				return nil, err
			}
		}
		if cfg.Direction.DoUp() {
			if err := pushIDs(ctx, client, sessionID, token, have, localByID); err != nil {
				return nil, err
			}
		}
	}
	return &SyncResult{Filter: filtersToAny(cfg.LocalFilter)}, nil
}

func fetchRelaySyncManifest(ctx context.Context, client *core.HTTPClient, id core.SessionID, token, snapshot []byte) ([]*relaysyncv2.PartitionSummary, error) {
	var result []*relaysyncv2.PartitionSummary
	var continuation []byte
	for {
		page, err := client.ManifestNext(ctx, id, token, &relaysyncv2.ManifestRequest{SessionId: id[:], SnapshotVersion: snapshot, Continuation: continuation, MaxPartitions: 64})
		if err != nil {
			return nil, fmt.Errorf("read RelaySync manifest: %w", err)
		}
		result = append(result, page.GetPartitions()...)
		if page.GetComplete() || len(page.GetContinuation()) == 0 {
			return result, nil
		}
		continuation = page.GetContinuation()
	}
}

func partitionForID(items map[core.PartitionKey][]core.EventMetadata, id [32]byte) (core.PartitionKey, []core.EventMetadata) {
	for partition, metadata := range items {
		candidate := core.PartitionIdentity(partition)
		if candidate == id {
			return partition, metadata
		}
	}
	return core.PartitionKey{}, nil
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
			if _, err := db.DbQueries().InsertEventBatch(ctx, events); err != nil {
				return fmt.Errorf("store pulled events: %w", err)
			}
		}
	}
	return nil
}

func pushLocalEvents(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, metadata []core.EventMetadata, events map[string]*nostr.Event) error {
	ids := make([]core.EventID, len(metadata))
	for i := range metadata {
		ids[i] = metadata[i].ID
	}
	return pushIDs(ctx, client, id, token, ids, events)
}

func pushIDs(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, ids []core.EventID, events map[string]*nostr.Event) error {
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		wireEvents := make([]*relaysyncv2.Event, 0, end-start)
		for _, eventID := range ids[start:end] {
			event := events[eventID.String()]
			if event == nil {
				continue
			}
			converted, err := coreEventFromNostr(event)
			if err != nil {
				return err
			}
			wireEvents = append(wireEvents, core.EventToProto(converted))
		}
		if len(wireEvents) == 0 {
			continue
		}
		if _, err := client.PushEvents(ctx, id, token, &relaysyncv2.PushEventsRequest{SessionId: id[:], OperationId: fmt.Sprintf("push-%d", start), Events: wireEvents, IdempotencyKey: []byte(fmt.Sprintf("push-%d", start))}); err != nil {
			return fmt.Errorf("push RelaySync events: %w", err)
		}
	}
	return nil
}

func coreEventFromNostr(event *nostr.Event) (core.Event, error) {
	id, err := core.ParseEventID(event.ID)
	if err != nil {
		return core.Event{}, err
	}
	pubkey, err := core.ParsePublicKey(event.PubKey)
	if err != nil {
		return core.Event{}, err
	}
	sig, err := core.ParseSignature(event.Sig)
	if err != nil {
		return core.Event{}, err
	}
	tags := make([]core.Tag, len(event.Tags))
	for i := range event.Tags {
		tags[i] = append(core.Tag(nil), event.Tags[i]...)
	}
	return core.Event{ID: id, PublicKey: pubkey, Signature: sig, CreatedAt: int64(event.CreatedAt), Kind: uint32(event.Kind), Tags: tags, Content: event.Content}, nil
}

func nostrEventFromCore(event core.Event) *nostr.Event {
	tags := make(nostr.Tags, len(event.Tags))
	for i := range event.Tags {
		tags[i] = append(nostr.Tag(nil), event.Tags[i]...)
	}
	return &nostr.Event{ID: event.ID.String(), PubKey: event.PublicKey.String(), Sig: event.Signature.String(), CreatedAt: nostr.Timestamp(event.CreatedAt), Kind: int(event.Kind), Tags: tags, Content: event.Content}
}
