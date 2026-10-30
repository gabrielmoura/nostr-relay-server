package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRelaySyncProgressLogsLifecycleAndCounters(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	progress := newRelaySyncProgress(zap.New(core), DirectionBoth)

	progress.start(3)
	progress.manifestPage(&relaysyncv2.Manifest{
		Partitions: []*relaysyncv2.PartitionSummary{{
			Partition: &relaysyncv2.Partition{EventCount: 4},
		}},
	})
	progress.manifestComplete()
	progress.partitionStarted(&relaysyncv2.PartitionSummary{
		Partition: &relaysyncv2.Partition{Bucket: 123, PrefixBits: 8, Prefix: []byte{0xab}, EventCount: 4},
	}, 2)
	progress.partitionComplete(time.Now(), 2, &relaysyncv2.PartitionSummary{
		Partition: &relaysyncv2.Partition{EventCount: 4},
	}, 1, 2, nil)
	progress.pullBatchStarted(2)
	progress.pullBatchCompleted(2, 2, time.Millisecond)
	progress.pushBatchStarted(1)
	progress.pushBatchCompleted(1, time.Millisecond)
	progress.complete(nil)

	entries := observed.All()
	if len(entries) != 10 {
		t.Fatalf("log entries = %d, want 10", len(entries))
	}
	if entries[0].Message != "RelaySync v2 sync started" {
		t.Fatalf("first message = %q", entries[0].Message)
	}
	if entries[1].Message != "RelaySync v2 manifest page discovered" {
		t.Fatalf("manifest page message = %q", entries[1].Message)
	}
	if got := entries[1].ContextMap()["remote_partitions_discovered"]; got != int64(1) {
		t.Fatalf("remote_partitions_discovered = %#v, want 1", got)
	}
	if entries[3].Message != "RelaySync v2 partition reconciliation started" {
		t.Fatalf("partition start message = %q", entries[3].Message)
	}
	if got := entries[3].ContextMap()["partition_prefix"]; got != "ab" {
		t.Fatalf("partition_prefix = %#v, want ab", got)
	}
	if entries[4].Message != "RelaySync v2 partition reconciliation completed" {
		t.Fatalf("partition completion message = %q", entries[4].Message)
	}
	partitionFields := entries[4].ContextMap()
	if partitionFields["have_count"] != int64(1) || partitionFields["need_count"] != int64(2) {
		t.Fatalf("partition counts = %#v, want have=1 need=2", partitionFields)
	}
	if entries[9].Message != "RelaySync v2 sync completed" {
		t.Fatalf("completion message = %q", entries[9].Message)
	}
	completionFields := entries[9].ContextMap()
	if completionFields["pulled_events_total"] != int64(2) || completionFields["sent_events_total"] != int64(1) {
		t.Fatalf("completion counters = %#v, want pulled=2 sent=1", completionFields)
	}
	for _, entry := range entries {
		if _, exists := entry.ContextMap()["event_id"]; exists {
			t.Fatalf("progress log %q must not include an event ID", entry.Message)
		}
	}
}

func TestRelaySyncFailureContinuationPolicy(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{name: "partition storage failure", ctx: context.Background(), err: partitionRelaySyncError("store_pull", errors.New("database unavailable")), want: true},
		{name: "remote invalid data", ctx: context.Background(), err: partitionRelaySyncError("pull", errors.New("invalid event")), want: true},
		{name: "unauthorized session", ctx: context.Background(), err: partitionRelaySyncError("reconcile", &core.HTTPResponseError{StatusCode: http.StatusUnauthorized}), want: false},
		{name: "conflicting operation", ctx: context.Background(), err: partitionRelaySyncError("push", &core.HTTPResponseError{StatusCode: http.StatusConflict}), want: false},
		{name: "expired snapshot", ctx: context.Background(), err: partitionRelaySyncError("pull", &core.HTTPResponseError{StatusCode: http.StatusGone}), want: false},
		{name: "invalid receipt", ctx: context.Background(), err: partitionRelaySyncError("push", fmt.Errorf("%w: mismatched receipt", errRelaySyncIntegrity)), want: false},
		{name: "cancelled", ctx: cancelled, err: partitionRelaySyncError("reconcile", errors.New("network")), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldContinueRelaySyncFailure(tt.ctx, tt.err); got != tt.want {
				t.Fatalf("shouldContinueRelaySyncFailure() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestRetryRelaySyncReadRetriesOnlyWhenResilienceIsEnabled(t *testing.T) {
	previous := config.Cfg
	t.Cleanup(func() { config.Cfg = previous })
	config.Cfg = &config.Config{RelaySync: config.RelaySyncConfig{Client: config.RelaySyncClientConfig{Resilience: config.RelaySyncResilienceConfig{Enabled: true, MaxAttempts: 2, InitialBackoffMilliseconds: 1, MaxBackoffMilliseconds: 1}}}}
	attempts := 0
	value, err := retryRelaySyncRead(context.Background(), "pull", func() (string, error) {
		attempts++
		if attempts == 1 {
			return "", &core.HTTPResponseError{StatusCode: http.StatusServiceUnavailable}
		}
		return "ok", nil
	})
	if err != nil || value != "ok" || attempts != 2 {
		t.Fatalf("retryRelaySyncRead() = %q, %v after %d attempts", value, err, attempts)
	}
}

func TestRelaySyncProgressFailureIncludesPhaseAndPartition(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	progress := newRelaySyncProgress(zap.New(core), DirectionDown)
	progress.partitionFailed("pull", &relaysyncv2.PartitionSummary{Partition: &relaysyncv2.Partition{Bucket: 7, PrefixBits: 8, Prefix: []byte{0xcd}}}, errors.New("invalid remote event"))

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["phase"] != "pull" || fields["partition_prefix"] != "cd" || fields["partitions_failed"] != int64(1) {
		t.Fatalf("failure fields = %#v", fields)
	}
}

func TestRelaySyncHTTPURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "websocket", input: "ws://relay.example/relay", want: "http://relay.example/relay"},
		{name: "secure websocket", input: "wss://relay.example", want: "https://relay.example"},
		{name: "http", input: "http://relay.example", want: "http://relay.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relaySyncHTTPURL(tt.input)
			if err != nil || got != tt.want {
				t.Fatalf("relaySyncHTTPURL(%q) = %q, %v; want %q", tt.input, got, err, tt.want)
			}
		})
	}
}

func TestRelaySyncCapabilityProbeFallsBackToLegacy(t *testing.T) {
	adapter, err := core.NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{1}})
	if err != nil {
		t.Fatal(err)
	}
	mux := httpTestMux(t, adapter)
	server := httptest.NewServer(mux)
	defer server.Close()

	result, available, err := runRelaySyncIfAvailable(context.Background(), &ConfSync{Remote: server.URL, Direction: DirectionDown})
	if err != nil {
		t.Fatal(err)
	}
	if available || result != nil {
		t.Fatalf("legacy capability must select NIP-77 fallback: available=%v result=%v", available, result)
	}
}

func TestRelaySyncCapabilityProbeSelectsV2(t *testing.T) {
	adapter, err := core.NewHTTPAdapter(&relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpTestMux(t, adapter))
	defer server.Close()

	_, available, err := runRelaySyncIfAvailable(context.Background(), &ConfSync{Remote: server.URL, Direction: DirectionDown, LocalFilter: []nostr.Filter{{}}})
	if !available {
		t.Fatalf("RelaySync v2 peer was not selected, err=%v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "database is not initialized") {
		t.Fatalf("v2 selection should enter RelaySync flow, got err=%v", err)
	}
}

func TestHandleRelaySyncPushReceipts(t *testing.T) {
	tests := []struct {
		name     string
		receipts []*relaysyncv2.EventReceipt
		events   []*relaysyncv2.Event
		wantErr  bool
	}{
		{
			name: "permanent rejection continues",
			receipts: []*relaysyncv2.EventReceipt{
				{Id: []byte{1}, Status: relaysyncv2.WriteStatus_WRITE_ACCEPTED},
				{Id: []byte{2}, Status: relaysyncv2.WriteStatus_WRITE_REJECTED_PERMANENT, Code: "invalid_event"},
			},
			events: []*relaysyncv2.Event{{Id: []byte{1}}, {Id: []byte{2}}},
		},
		{
			name:     "temporary rejection fails",
			receipts: []*relaysyncv2.EventReceipt{{Id: []byte{1}, Status: relaysyncv2.WriteStatus_WRITE_FAILED_TEMPORARY, Code: "storage_unavailable"}},
			events:   []*relaysyncv2.Event{{Id: []byte{1}}},
			wantErr:  true,
		},
		{
			name:     "missing receipt fails",
			receipts: []*relaysyncv2.EventReceipt{{Id: []byte{1}, Status: relaysyncv2.WriteStatus_WRITE_ACCEPTED}},
			events:   []*relaysyncv2.Event{{Id: []byte{1}}, {Id: []byte{2}}},
			wantErr:  true,
		},
		{
			name:     "mismatched receipt id fails",
			receipts: []*relaysyncv2.EventReceipt{{Id: []byte{2}, Status: relaysyncv2.WriteStatus_WRITE_ACCEPTED}},
			events:   []*relaysyncv2.Event{{Id: []byte{1}}},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleRelaySyncPushReceipts(tt.receipts, tt.events)
			if (err != nil) != tt.wantErr {
				t.Fatalf("handleRelaySyncPushReceipts() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func httpTestMux(t *testing.T, adapter *core.HTTPAdapter) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	adapter.Register(mux)
	return mux
}
