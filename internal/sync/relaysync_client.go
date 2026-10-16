package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	randv2 "math/rand/v2"
	"net"
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

const relaySyncProgressLogInterval = 25
const relaySyncSessionCleanupTimeout = 10 * time.Second

var errRelaySyncIntegrity = errors.New("RelaySync response integrity failure")

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

func runRelaySyncHTTP(ctx context.Context, client *core.HTTPClient, cfg *ConfSync) (result *SyncResult, err error) {
	progress := newRelaySyncProgress(log.Logger, cfg.Direction)
	defer func() {
		progress.complete(err)
	}()

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
	progress.start(len(localPartitions))

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
		cleanupCtx, cancel := context.WithTimeout(context.Background(), relaySyncSessionCleanupTimeout)
		defer cancel()
		if err != nil {
			if cleanupErr := client.DeleteSession(cleanupCtx, sessionID, token); cleanupErr != nil && log.Logger != nil {
				log.Logger.Warn("failed to abandon unfinished RelaySync session", zap.Error(cleanupErr))
			}
			return
		}
		if cleanupErr := client.CompleteSession(cleanupCtx, sessionID, token); cleanupErr != nil {
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
	partialFailures := &relaySyncPartialFailures{}
	manifestComplete := true
	for result := range streamRelaySyncManifest(manifestCtx, client, sessionID, token, snapshotVersion, 10) {
		if result.err != nil {
			manifestErr := fmt.Errorf("read RelaySync manifest: %w", result.err)
			if !shouldContinueRelaySyncFailure(ctx, manifestErr) || !relaySyncResilienceEnabled() {
				return nil, manifestErr
			}
			manifestComplete = false
			partialFailures.add("manifest", nil, manifestErr)
			progress.partitionFailed("manifest", nil, manifestErr)
			break
		}
		progress.manifestPage(result.page)
		for index, summary := range result.page.GetPartitions() {
			if summary == nil || summary.GetPartition() == nil || len(summary.GetPartition().GetId()) != 32 {
				invalidErr := fmt.Errorf("remote returned invalid RelaySync partition at manifest index %d", index)
				if !relaySyncResilienceEnabled() {
					return nil, invalidErr
				}
				partialFailures.add("manifest", nil, invalidErr)
				progress.partitionFailed("manifest", nil, invalidErr)
				continue
			}
			rawID := string(summary.GetPartition().GetId())
			seenRemote[rawID] = struct{}{}
			partition, exists := localPartitions[rawID]
			var localItems []core.EventMetadata
			if exists {
				localItems, err = readPartitionMetadata(ctx, localSnapshot, partition)
				if err != nil {
					metadataErr := partitionRelaySyncError("local_metadata", err)
					if !shouldContinueRelaySyncFailure(ctx, metadataErr) || !relaySyncResilienceEnabled() {
						return nil, metadataErr
					}
					partialFailures.add("local_metadata", summary, metadataErr)
					progress.partitionFailed("local_metadata", summary, metadataErr)
					continue
				}
			}
			progress.partitionStarted(summary, len(localItems))
			var partitionID [32]byte
			copy(partitionID[:], summary.GetPartition().GetId())
			if err := syncRelayPartition(ctx, client, cfg, sessionID, token, localSnapshot, partitionID, localItems, summary, progress); err != nil {
				if !shouldContinueRelaySyncFailure(ctx, err) || !relaySyncResilienceEnabled() {
					return nil, err
				}
				partialFailures.add(relaySyncFailurePhase(err), summary, err)
				progress.partitionFailed(relaySyncFailurePhase(err), summary, err)
				continue
			}
		}
	}
	progress.manifestComplete()
	if cfg.Direction.DoUp() && manifestComplete {
		for rawID, partition := range localPartitions {
			if _, exists := seenRemote[rawID]; exists {
				continue
			}
			localSummary := relaySyncSummaryForLocalPartition(partition)
			localItems, err := readPartitionMetadata(ctx, localSnapshot, partition)
			if err != nil {
				metadataErr := partitionRelaySyncError("local_metadata", err)
				if !shouldContinueRelaySyncFailure(ctx, metadataErr) || !relaySyncResilienceEnabled() {
					return nil, metadataErr
				}
				partialFailures.add("local_metadata", localSummary, metadataErr)
				progress.partitionFailed("local_metadata", localSummary, metadataErr)
				continue
			}
			ids := metadataIDs(localItems)
			var partitionID [32]byte
			copy(partitionID[:], rawID)
			if err := pushSnapshotIDs(ctx, client, sessionID, token, localSnapshot, partitionID, ids, progress); err != nil {
				if !shouldContinueRelaySyncFailure(ctx, err) || !relaySyncResilienceEnabled() {
					return nil, err
				}
				partialFailures.add(relaySyncFailurePhase(err), localSummary, err)
				progress.partitionFailed(relaySyncFailurePhase(err), localSummary, err)
			}
		}
	}
	if !manifestComplete {
		progress.info("RelaySync v2 skipped local-only partitions because manifest was incomplete")
	}
	if partialFailures.len() > 0 {
		return nil, partialFailures
	}
	return &SyncResult{Filter: filtersToAny(cfg.LocalFilter)}, nil
}

func relaySyncSummaryForLocalPartition(partition core.PartitionKey) *relaysyncv2.PartitionSummary {
	identity := core.PartitionIdentity(partition)
	return &relaysyncv2.PartitionSummary{Partition: &relaysyncv2.Partition{
		Id:               identity[:],
		AlgorithmVersion: partition.AlgorithmVersion,
		Bucket:           partition.Bucket,
		PrefixBits:       uint32(partition.PrefixBits),
		Prefix:           append([]byte(nil), partition.Prefix[:]...),
	}}
}

func relaySyncResilienceEnabled() bool {
	return config.Cfg != nil && config.Cfg.RelaySync.Client.Resilience.Enabled
}

type relaySyncPartitionError struct {
	phase string
	err   error
}

func (e *relaySyncPartitionError) Error() string {
	return fmt.Sprintf("RelaySync %s: %v", e.phase, e.err)
}
func (e *relaySyncPartitionError) Unwrap() error { return e.err }

func partitionRelaySyncError(phase string, err error) error {
	if err == nil {
		return nil
	}
	return &relaySyncPartitionError{phase: phase, err: err}
}

func relaySyncFailurePhase(err error) string {
	var partitionErr *relaySyncPartitionError
	if errors.As(err, &partitionErr) {
		return partitionErr.phase
	}
	return "partition"
}

// Authentication, session lifecycle and receipt-integrity failures mean the
// client cannot safely reason about the rest of this finite cycle. Other
// partition-scoped failures are recorded as partial progress when resilience
// is enabled.
func shouldContinueRelaySyncFailure(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, errRelaySyncIntegrity) {
		return false
	}
	var responseErr *core.HTTPResponseError
	if errors.As(err, &responseErr) {
		switch responseErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict, http.StatusGone, http.StatusNotFound:
			return false
		}
	}
	return true
}

func retryRelaySyncRead[T any](ctx context.Context, phase string, action func() (T, error)) (T, error) {
	var zero T
	maxAttempts := 1
	initialBackoff := 250 * time.Millisecond
	maxBackoff := 5 * time.Second
	if config.Cfg != nil && config.Cfg.RelaySync.Client.Resilience.Enabled {
		resilience := config.Cfg.RelaySync.Client.Resilience
		if resilience.MaxAttempts > 0 {
			maxAttempts = resilience.MaxAttempts
		}
		if resilience.InitialBackoffMilliseconds > 0 {
			initialBackoff = time.Duration(resilience.InitialBackoffMilliseconds) * time.Millisecond
		}
		if resilience.MaxBackoffMilliseconds >= resilience.InitialBackoffMilliseconds && resilience.MaxBackoffMilliseconds > 0 {
			maxBackoff = time.Duration(resilience.MaxBackoffMilliseconds) * time.Millisecond
		}
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		value, err := action()
		if err == nil || !relaySyncRetryable(ctx, err) || attempt == maxAttempts {
			return value, err
		}
		backoff := min(initialBackoff<<(attempt-1), maxBackoff)
		delay := time.Duration(randv2.Int64N(int64(backoff) + 1))
		if log.Logger != nil {
			log.Logger.Warn("retrying RelaySync v2 read operation", zap.String("phase", phase), zap.Int("attempt", attempt+1), zap.Duration("delay", delay), zap.Error(err))
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, nil
}

func relaySyncRetryable(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) {
		return false
	}
	var responseErr *core.HTTPResponseError
	if errors.As(err, &responseErr) {
		return responseErr.StatusCode == http.StatusRequestTimeout || responseErr.StatusCode == http.StatusTooEarly || responseErr.StatusCode == http.StatusTooManyRequests || responseErr.StatusCode >= http.StatusInternalServerError
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkErr net.Error
	return errors.As(err, &networkErr) && (networkErr.Timeout() || networkErr.Temporary())
}

type relaySyncPartialFailure struct {
	phase     string
	partition string
	err       error
}

type relaySyncPartialFailures struct{ failures []relaySyncPartialFailure }

func (f *relaySyncPartialFailures) add(phase string, summary *relaysyncv2.PartitionSummary, err error) {
	partition := ""
	if summary != nil && summary.GetPartition() != nil {
		partition = hex.EncodeToString(summary.GetPartition().GetId())
	}
	f.failures = append(f.failures, relaySyncPartialFailure{phase: phase, partition: partition, err: err})
}
func (f *relaySyncPartialFailures) len() int { return len(f.failures) }
func (f *relaySyncPartialFailures) Error() string {
	return fmt.Sprintf("RelaySync v2 completed with %d unresolved partition failure(s)", len(f.failures))
}

// relaySyncProgress keeps one bounded, operator-facing summary for a RelaySync
// v2 run. It deliberately records counts rather than event identifiers,
// payloads, session tokens, or remote-provided messages.
type relaySyncProgress struct {
	logger    *zap.Logger
	direction Direction
	startedAt time.Time

	localPartitions     int
	manifestPages       int
	remotePartitions    int
	completedPartitions int
	failedPartitions    int
	pullBatches         int
	pulledEvents        int
	pushBatches         int
	pushedEvents        int
}

func newRelaySyncProgress(logger *zap.Logger, direction Direction) *relaySyncProgress {
	return &relaySyncProgress{logger: logger, direction: direction, startedAt: time.Now()}
}

func (p *relaySyncProgress) start(localPartitions int) {
	p.localPartitions = localPartitions
	p.info("RelaySync v2 sync started",
		zap.String("direction", string(p.direction)),
		zap.Int("local_partition_count", localPartitions),
	)
}

func (p *relaySyncProgress) manifestPage(page *relaysyncv2.Manifest) {
	partitions := len(page.GetPartitions())
	p.manifestPages++
	p.remotePartitions += partitions
	p.debug("RelaySync v2 manifest page discovered",
		zap.Int("manifest_page", p.manifestPages),
		zap.Int("partition_count", partitions),
		zap.Int("remote_partitions_discovered", p.remotePartitions),
		zap.Bool("complete", page.GetComplete()),
	)
}

func (p *relaySyncProgress) manifestComplete() {
	p.info("RelaySync v2 manifest discovery completed",
		zap.Int("manifest_pages", p.manifestPages),
		zap.Int("remote_partition_count", p.remotePartitions),
		zap.Duration("elapsed", time.Since(p.startedAt)),
	)
}

func (p *relaySyncProgress) partitionStarted(remote *relaysyncv2.PartitionSummary, localCount int) {
	partition := remote.GetPartition()
	p.debug("RelaySync v2 partition reconciliation started",
		zap.Int64("partition_bucket", partition.GetBucket()),
		zap.Uint32("partition_prefix_bits", partition.GetPrefixBits()),
		zap.String("partition_prefix", hex.EncodeToString(partition.GetPrefix())),
		zap.Int("local_event_count", localCount),
		zap.Uint64("remote_event_count", partition.GetEventCount()),
		zap.Int("partitions_completed", p.completedPartitions),
		zap.Int("remote_partitions_discovered", p.remotePartitions),
	)
}

func (p *relaySyncProgress) partitionComplete(startedAt time.Time, localCount int, remote *relaysyncv2.PartitionSummary, have, need int, err error) {
	partition := remote.GetPartition()
	fields := []zap.Field{
		zap.Int64("partition_bucket", partition.GetBucket()),
		zap.Uint32("partition_prefix_bits", partition.GetPrefixBits()),
		zap.String("partition_prefix", hex.EncodeToString(partition.GetPrefix())),
		zap.Int("local_event_count", localCount),
		zap.Uint64("remote_event_count", partition.GetEventCount()),
		zap.Int("have_count", have),
		zap.Int("need_count", need),
		zap.Duration("elapsed", time.Since(startedAt)),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		p.error("RelaySync v2 partition reconciliation failed", fields...)
		return
	}
	p.completedPartitions++
	fields = append(fields, zap.Int("partitions_completed", p.completedPartitions))
	p.debug("RelaySync v2 partition reconciliation completed", fields...)
	if p.completedPartitions%relaySyncProgressLogInterval == 0 {
		p.info("RelaySync v2 sync progress checkpoint", fields...)
	}
}

func (p *relaySyncProgress) partitionFailed(phase string, remote *relaysyncv2.PartitionSummary, err error) {
	p.failedPartitions++
	fields := []zap.Field{
		zap.String("phase", phase),
		zap.Int("partitions_failed", p.failedPartitions),
		zap.Error(err),
	}
	if remote != nil && remote.GetPartition() != nil {
		partition := remote.GetPartition()
		fields = append(fields,
			zap.Int64("partition_bucket", partition.GetBucket()),
			zap.Uint32("partition_prefix_bits", partition.GetPrefixBits()),
			zap.String("partition_prefix", hex.EncodeToString(partition.GetPrefix())),
		)
	}
	p.error("RelaySync v2 partition skipped after recoverable failure", fields...)
}

func (p *relaySyncProgress) pullBatchStarted(batchSize int) {
	p.debug("RelaySync v2 pull batch started",
		zap.Int("batch_size", batchSize),
		zap.Int("pull_batch", p.pullBatches+1),
		zap.Int("pulled_events_total", p.pulledEvents),
	)
}

func (p *relaySyncProgress) pullBatchCompleted(batchSize, received int, elapsed time.Duration) {
	p.pullBatches++
	p.pulledEvents += received
	p.debug("RelaySync v2 pull batch completed",
		zap.Int("requested_event_count", batchSize),
		zap.Int("received_event_count", received),
		zap.Int("pull_batch", p.pullBatches),
		zap.Int("pulled_events_total", p.pulledEvents),
		zap.Duration("elapsed", elapsed),
	)
}

func (p *relaySyncProgress) pullEventRejected(batchOffset int, code string, err error) {
	p.error("RelaySync v2 pulled event rejected",
		zap.String("phase", "validate_remote_event"),
		zap.Int("batch_offset", batchOffset),
		zap.String("code", code),
		zap.Error(err),
	)
}

func (p *relaySyncProgress) pushBatchStarted(batchSize int) {
	p.debug("RelaySync v2 push batch started",
		zap.Int("batch_size", batchSize),
		zap.Int("push_batch", p.pushBatches+1),
		zap.Int("sent_events_total", p.pushedEvents),
	)
}

func (p *relaySyncProgress) pushBatchCompleted(batchSize int, elapsed time.Duration) {
	p.pushBatches++
	p.pushedEvents += batchSize
	p.debug("RelaySync v2 push batch completed",
		zap.Int("sent_event_count", batchSize),
		zap.Int("push_batch", p.pushBatches),
		zap.Int("sent_events_total", p.pushedEvents),
		zap.Duration("elapsed", elapsed),
	)
}

func (p *relaySyncProgress) complete(err error) {
	fields := []zap.Field{
		zap.Int("local_partition_count", p.localPartitions),
		zap.Int("manifest_pages", p.manifestPages),
		zap.Int("remote_partition_count", p.remotePartitions),
		zap.Int("partitions_completed", p.completedPartitions),
		zap.Int("partitions_failed", p.failedPartitions),
		zap.Int("pull_batches", p.pullBatches),
		zap.Int("pulled_events_total", p.pulledEvents),
		zap.Int("push_batches", p.pushBatches),
		zap.Int("sent_events_total", p.pushedEvents),
		zap.Duration("elapsed", time.Since(p.startedAt)),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		p.error("RelaySync v2 sync failed", fields...)
		return
	}
	p.info("RelaySync v2 sync completed", fields...)
}

func (p *relaySyncProgress) info(message string, fields ...zap.Field) {
	if p.logger != nil {
		p.logger.Info(message, fields...)
	}
}

func (p *relaySyncProgress) error(message string, fields ...zap.Field) {
	if p.logger != nil {
		p.logger.Error(message, fields...)
	}
}

func (p *relaySyncProgress) debug(message string, fields ...zap.Field) {
	if p.logger != nil {
		p.logger.Debug(message, fields...)
	}
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
				page, err := retryRelaySyncRead(ctx, "manifest", func() (*relaysyncv2.Manifest, error) {
					return client.ManifestNext(ctx, id, token, &relaysyncv2.ManifestRequest{SessionId: id[:], SnapshotVersion: snapshot, Continuation: continuation, MaxPartitions: 1})
				})
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

func syncRelayPartition(ctx context.Context, client *core.HTTPClient, cfg *ConfSync, sessionID core.SessionID, token []byte, localSnapshot core.Snapshot, partitionID [32]byte, localItems []core.EventMetadata, remoteSummary *relaysyncv2.PartitionSummary, progress *relaySyncProgress) (err error) {
	startedAt := time.Now()
	var have, need []core.EventID
	defer func() {
		progress.partitionComplete(startedAt, len(localItems), remoteSummary, len(have), len(need), err)
	}()

	localDigest, err := core.DigestMetadata(localItems)
	if err != nil {
		return partitionRelaySyncError("local_metadata", err)
	}
	if string(localDigest[:]) == string(remoteSummary.GetDigest()) {
		return nil
	}
	have, need, err = reconcilePartition(ctx, client, sessionID, token, partitionID, localItems)
	if err != nil {
		return partitionRelaySyncError("reconcile", err)
	}
	if cfg.Direction.DoDown() {
		if err := pullAndStoreEvents(ctx, client, sessionID, token, need, progress); err != nil {
			return partitionRelaySyncError("pull", err)
		}
	}
	if cfg.Direction.DoUp() {
		if err := pushSnapshotIDs(ctx, client, sessionID, token, localSnapshot, partitionID, have, progress); err != nil {
			return partitionRelaySyncError("push", err)
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
		response, err := retryRelaySyncRead(ctx, "reconcile", func() (*relaysyncv2.ReconcileResponse, error) {
			return client.Reconcile(ctx, id, token, &relaysyncv2.ReconcileRequest{SessionId: id[:], PartitionId: partitionID[:], NegentropyPayload: frame, Round: round})
		})
		if err != nil {
			return nil, nil, fmt.Errorf("reconcile partition: %w", err)
		}
		if !bytes.Equal(response.GetPartitionId(), partitionID[:]) || response.GetRound() != round {
			return nil, nil, fmt.Errorf("%w: reconcile response does not match partition and round", errRelaySyncIntegrity)
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

func pullAndStoreEvents(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, ids []core.EventID, progress *relaySyncProgress) error {
	totalRejected := 0
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		wireIDs := make([][]byte, end-start)
		for i := range wireIDs {
			wireIDs[i] = append([]byte(nil), ids[start+i][:]...)
		}
		startedAt := time.Now()
		progress.pullBatchStarted(len(wireIDs))
		response, err := retryRelaySyncRead(ctx, "pull", func() (*relaysyncv2.PullEventsResponse, error) {
			return client.PullEvents(ctx, id, token, &relaysyncv2.PullEventsRequest{SessionId: id[:], OperationId: fmt.Sprintf("pull-%d", start), EventIds: wireIDs, MaxEvents: uint32(len(wireIDs))})
		})
		if err != nil {
			return fmt.Errorf("pull RelaySync events: %w", err)
		}
		requested := make(map[string]struct{}, len(ids[start:end]))
		for _, requestedID := range ids[start:end] {
			requested[string(requestedID[:])] = struct{}{}
		}
		responded := make(map[string]struct{}, len(response.GetEvents())+len(response.GetMissingEventIds()))
		events := make([]*nostr.Event, 0, len(response.GetEvents()))
		invalidEvents := 0
		for _, wire := range response.GetEvents() {
			if wire == nil || len(wire.GetId()) != 32 {
				return fmt.Errorf("%w: pull response contains an event without a usable id", errRelaySyncIntegrity)
			}
			key := string(wire.GetId())
			if _, ok := requested[key]; !ok {
				return fmt.Errorf("%w: pull response contains an unsolicited event", errRelaySyncIntegrity)
			}
			if _, duplicate := responded[key]; duplicate {
				return fmt.Errorf("%w: pull response contains a duplicate event result", errRelaySyncIntegrity)
			}
			responded[key] = struct{}{}
			event, err := core.EventFromProto(wire, core.DefaultLimits())
			if err != nil {
				invalidEvents++
				progress.pullEventRejected(start, "invalid_event", err)
				continue
			}
			if err := event.Validate(core.DefaultLimits()); err != nil {
				invalidEvents++
				progress.pullEventRejected(start, "invalid_signature", err)
				continue
			}
			if string(event.ID[:]) != key {
				return fmt.Errorf("%w: pull response event id changed during decode", errRelaySyncIntegrity)
			}
			events = append(events, nostrEventFromCore(event))
		}
		for _, missingID := range response.GetMissingEventIds() {
			if len(missingID) != 32 {
				return fmt.Errorf("%w: pull response contains an invalid missing event id", errRelaySyncIntegrity)
			}
			key := string(missingID)
			if _, ok := requested[key]; !ok {
				return fmt.Errorf("%w: pull response contains an unsolicited missing event id", errRelaySyncIntegrity)
			}
			if _, duplicate := responded[key]; duplicate {
				return fmt.Errorf("%w: pull response contains a duplicate event result", errRelaySyncIntegrity)
			}
			responded[key] = struct{}{}
		}
		if len(responded) != len(requested) {
			return fmt.Errorf("%w: pull response does not account for every requested event", errRelaySyncIntegrity)
		}
		if len(events) > 0 {
			inserted, err := internaldb.DbQueries.InsertEventBatch(ctx, events)
			if err != nil {
				return fmt.Errorf("store pulled events: %w", err)
			}
			if len(inserted.RejectedEventIDs) > 0 {
				invalidEvents += len(inserted.RejectedEventIDs)
				progress.pullEventRejected(start, "local_storage_rejected", fmt.Errorf("%d events rejected by local storage", len(inserted.RejectedEventIDs)))
			}
		}
		progress.pullBatchCompleted(len(wireIDs), len(events), time.Since(startedAt))
		totalRejected += invalidEvents
	}
	if totalRejected > 0 {
		return fmt.Errorf("pull RelaySync events: %d invalid or rejected event(s)", totalRejected)
	}
	return nil
}

func pushSnapshotIDs(ctx context.Context, client *core.HTTPClient, id core.SessionID, token []byte, snapshot core.Snapshot, partitionID [32]byte, ids []core.EventID, progress *relaySyncProgress) error {
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
		startedAt := time.Now()
		progress.pushBatchStarted(len(wireEvents))
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
		progress.pushBatchCompleted(len(wireEvents), time.Since(startedAt))
	}
	return nil
}

// handleRelaySyncPushReceipts makes permanent remote policy or validation
// decisions visible without aborting an otherwise valid finite sync cycle.
// A temporary decision remains fatal so it can be retried instead of being
// silently treated as convergence.
func handleRelaySyncPushReceipts(receipts []*relaysyncv2.EventReceipt, events []*relaysyncv2.Event) error {
	if len(receipts) != len(events) {
		return fmt.Errorf("%w: push RelaySync events received %d receipts for %d events", errRelaySyncIntegrity, len(receipts), len(events))
	}
	for index, receipt := range receipts {
		if receipt == nil {
			return fmt.Errorf("%w: push RelaySync events received empty receipt", errRelaySyncIntegrity)
		}
		if !bytes.Equal(receipt.GetId(), events[index].GetId()) {
			return fmt.Errorf("%w: push RelaySync events receipt id does not match sent event", errRelaySyncIntegrity)
		}
		switch receipt.GetStatus() {
		case relaysyncv2.WriteStatus_WRITE_ACCEPTED, relaysyncv2.WriteStatus_WRITE_ALREADY_EXISTS:
			continue
		case relaysyncv2.WriteStatus_WRITE_REJECTED_PERMANENT:
			if log.Logger != nil {
				log.Logger.Warn("RelaySync event permanently rejected by remote",
					zap.String("event_id", hex.EncodeToString(receipt.GetId())),
					zap.String("code", receipt.GetCode()),
					zap.String("reason", receipt.GetMessage()),
				)
			}
			continue
		case relaysyncv2.WriteStatus_WRITE_FAILED_TEMPORARY:
			return fmt.Errorf("push RelaySync events: remote temporary failure (%s)", receipt.GetCode())
		default:
			return fmt.Errorf("%w: push RelaySync events remote returned invalid receipt status (%s)", errRelaySyncIntegrity, receipt.GetStatus().String())
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
