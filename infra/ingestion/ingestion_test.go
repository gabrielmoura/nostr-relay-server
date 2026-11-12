package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dmimport"
	"github.com/nbd-wtf/go-nostr"
)

func TestDeliverAfterGroupProjectionDeliversOnlyAfterSuccessfulProjection(t *testing.T) {
	steps := make([]string, 0, 2)
	err := deliverAfterGroupProjection(context.Background(), &nostr.Event{ID: "private-group-event"}, func(context.Context, *nostr.Event) error {
		steps = append(steps, "project")
		return nil
	}, func() {
		steps = append(steps, "deliver")
	})
	if err != nil {
		t.Fatalf("deliverAfterGroupProjection() error = %v", err)
	}
	if got, want := len(steps), 2; got != want || steps[0] != "project" || steps[1] != "deliver" {
		t.Fatalf("delivery order = %#v, want [project deliver]", steps)
	}
}

func TestDeliverAfterGroupProjectionSuppressesPrivateEventOnProjectionFailure(t *testing.T) {
	delivered := false
	projectionErr := errors.New("group projection unavailable")
	err := deliverAfterGroupProjection(context.Background(), &nostr.Event{ID: "private-group-event"}, func(context.Context, *nostr.Event) error {
		return projectionErr
	}, func() {
		delivered = true
	})
	if !errors.Is(err, projectionErr) {
		t.Fatalf("deliverAfterGroupProjection() error = %v, want %v", err, projectionErr)
	}
	if delivered {
		t.Fatal("private event was delivered after group projection failure")
	}
}

func TestDurableReceiptDoesNotAcknowledgeFailedProjection(t *testing.T) {
	receipt := durableReceipt(batchResult{Inserted: 1, Uncertain: 1}, errors.New("projection failed after persistence"))
	if !receipt.Uncertain || receipt.Inserted || receipt.Duplicate || receipt.Rejected {
		t.Fatalf("failed projection returned a final receipt: %#v", receipt)
	}
}

func TestPushDMImportWaitsForQueueCapacityUntilContextExpires(t *testing.T) {
	queue := make(chan queueItem, 1)
	queue <- queueItem{}
	restore := setIngestionStateForTest(t, queue)
	defer restore()

	event := &nostr.Event{ID: "event", Kind: nostr.KindEncryptedDirectMessage, PubKey: "owner"}
	admission, ok := dmimport.Authorize("owner", event)
	if !ok {
		t.Fatal("expected valid admission")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	receipt, accepted := PushDMImport(ctx, event, admission)
	if accepted || receipt.Uncertain || receipt.Reason != "error: import queue wait timed out" {
		t.Fatalf("PushDMImport() = (%#v, %t)", receipt, accepted)
	}
}

func TestPushDMImportTimeoutAfterQueueingIsUncertain(t *testing.T) {
	queue := make(chan queueItem, 1)
	restore := setIngestionStateForTest(t, queue)
	defer restore()

	event := &nostr.Event{ID: "event", Kind: nostr.KindEncryptedDirectMessage, PubKey: "owner"}
	admission, ok := dmimport.Authorize("owner", event)
	if !ok {
		t.Fatal("expected valid admission")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type response struct {
		receipt Receipt
		ok      bool
	}
	result := make(chan response, 1)
	go func() {
		receipt, accepted := PushDMImport(ctx, event, admission)
		result <- response{receipt: receipt, ok: accepted}
	}()

	select {
	case <-queue:
	case <-time.After(time.Second):
		t.Fatal("event was not queued")
	}
	cancel()

	select {
	case got := <-result:
		if got.ok || !got.receipt.Uncertain || got.receipt.Reason != "error: import outcome uncertain (timed out after queueing)" {
			t.Fatalf("PushDMImport() = (%#v, %t)", got.receipt, got.ok)
		}
	case <-time.After(time.Second):
		t.Fatal("PushDMImport did not return after cancellation")
	}
}

func TestPushDurableTimeoutAfterQueueingIsUncertain(t *testing.T) {
	queue := make(chan queueItem, 1)
	restore := setIngestionStateForTest(t, queue)
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type response struct {
		receipt Receipt
		ok      bool
	}
	result := make(chan response, 1)
	go func() {
		receipt, accepted := PushDurable(ctx, &nostr.Event{ID: "event"})
		result <- response{receipt, accepted}
	}()
	select {
	case <-queue:
	case <-time.After(time.Second):
		t.Fatal("event was not queued")
	}
	cancel()
	select {
	case got := <-result:
		if got.ok || !got.receipt.Uncertain || got.receipt.Reason != "error: event outcome uncertain (timed out after queueing)" {
			t.Fatalf("PushDurable() = (%#v, %t)", got.receipt, got.ok)
		}
	case <-time.After(time.Second):
		t.Fatal("PushDurable did not return after cancellation")
	}
}

func TestDurableCompletionRunsAfterCallerCancellation(t *testing.T) {
	queue := make(chan queueItem, 1)
	restore := setIngestionStateForTest(t, queue)
	defer restore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completed := make(chan Receipt, 1)
	type outcome struct {
		receipt          Receipt
		accepted, queued bool
	}
	result := make(chan outcome, 1)
	go func() {
		receipt, accepted, queued := PushDurableWithCompletion(ctx, &nostr.Event{ID: "event"}, func(r Receipt) { completed <- r })
		result <- outcome{receipt, accepted, queued}
	}()
	var item queueItem
	select {
	case item = <-queue:
	case <-time.After(time.Second):
		t.Fatal("event was not queued")
	}
	cancel()
	select {
	case got := <-result:
		if got.accepted || !got.queued || !got.receipt.Uncertain {
			t.Fatalf("outcome=%#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("caller did not return")
	}
	completeDurable(item, Receipt{Inserted: true})
	select {
	case got := <-completed:
		if !got.Inserted {
			t.Fatalf("receipt=%#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("completion did not run")
	}
}

func TestMaterializeImportedEventOnlyProfiles(t *testing.T) {
	previous := upsertImportedProfile
	defer func() { upsertImportedProfile = previous }()

	var got *nostr.Event
	upsertImportedProfile = func(_ context.Context, event *nostr.Event) error {
		got = event
		return nil
	}

	materializeImportedEvent(context.Background(), &nostr.Event{Kind: nostr.KindTextNote})
	if got != nil {
		t.Fatalf("non-profile event was materialized: %#v", got)
	}

	profile := &nostr.Event{ID: "profile", Kind: nostr.KindProfileMetadata}
	materializeImportedEvent(context.Background(), profile)
	if got != profile {
		t.Fatalf("materialized event = %#v, want %#v", got, profile)
	}
}

func setIngestionStateForTest(t *testing.T, queue chan queueItem) func() {
	t.Helper()
	startMutex.Lock()
	previousQueue := eventsChan
	previousStopping := stopping
	previousStarted := started
	eventsChan = queue
	stopping = make(chan struct{})
	started = true
	startMutex.Unlock()

	return func() {
		startMutex.Lock()
		eventsChan = previousQueue
		stopping = previousStopping
		started = previousStarted
		startMutex.Unlock()
	}
}

func TestApplyDMImportBatchResultReportsEachEventOutcome(t *testing.T) {
	inserted := &nostr.Event{ID: "inserted"}
	duplicate := &nostr.Event{ID: "duplicate"}
	rejected := &nostr.Event{ID: "rejected"}
	receipts := make([]Receipt, 4)

	result := applyDMImportBatchResult(
		[]*nostr.Event{inserted, duplicate, rejected},
		[]int{2, 0, 3},
		dbstore.BatchInsertResult{
			InsertedIDs:      []string{inserted.ID},
			RejectedEventIDs: []string{rejected.ID},
		},
		batchResult{},
		receipts,
		true,
	)

	if result.Inserted != 1 || result.Duplicates != 1 || result.Rejected != 1 {
		t.Fatalf("batch result = %#v, want one result of each type", result)
	}
	if !receipts[2].Inserted || receipts[2].Duplicate || receipts[2].Reason != "" {
		t.Fatalf("inserted receipt = %#v", receipts[2])
	}
	if !receipts[0].Duplicate || receipts[0].Inserted || receipts[0].Reason != "" {
		t.Fatalf("duplicate receipt = %#v", receipts[0])
	}
	if receipts[3].Inserted || receipts[3].Duplicate || receipts[3].Reason == "" {
		t.Fatalf("rejected receipt = %#v", receipts[3])
	}
	if receipts[1] != (Receipt{}) {
		t.Fatalf("non-import receipt = %#v, want zero value", receipts[1])
	}
}

func TestApplyDMImportBatchResultMarksUnknownPartialBatchOutcomesUncertain(t *testing.T) {
	inserted := &nostr.Event{ID: "inserted"}
	unknown := &nostr.Event{ID: "unknown"}
	receipts := make([]Receipt, 2)

	result := applyDMImportBatchResult(
		[]*nostr.Event{inserted, unknown},
		[]int{0, 1},
		dbstore.BatchInsertResult{InsertedIDs: []string{inserted.ID}},
		batchResult{},
		receipts,
		false,
	)

	if result.Inserted != 1 || result.Duplicates != 0 || result.Rejected != 0 || result.Uncertain != 1 {
		t.Fatalf("partial batch result = %#v", result)
	}
	if !receipts[0].Inserted || receipts[0].Uncertain {
		t.Fatalf("known inserted receipt = %#v", receipts[0])
	}
	if !receipts[1].Uncertain || receipts[1].Duplicate || receipts[1].Inserted || receipts[1].Reason == "" {
		t.Fatalf("unknown receipt = %#v", receipts[1])
	}
}

func TestApplyDMImportBatchResultTreatsRepeatedIDInOneBatchAsDuplicate(t *testing.T) {
	first := &nostr.Event{ID: "same"}
	second := &nostr.Event{ID: "same"}
	receipts := make([]Receipt, 2)

	result := applyDMImportBatchResult(
		[]*nostr.Event{first, second},
		[]int{0, 1},
		dbstore.BatchInsertResult{InsertedIDs: []string{first.ID}},
		batchResult{},
		receipts,
		true,
	)
	if result.Inserted != 1 || result.Duplicates != 1 {
		t.Fatalf("result = %#v, want one insertion and one duplicate", result)
	}
	if !receipts[0].Inserted || !receipts[1].Duplicate {
		t.Fatalf("receipts = %#v", receipts)
	}
}

func TestPrepareEventForStorageReplacesOlderAddressableKind30443(t *testing.T) {
	prevQuery := queryFirstStoredEvent
	prevDelete := deleteStoredEventByID
	t.Cleanup(func() {
		queryFirstStoredEvent = prevQuery
		deleteStoredEventByID = prevDelete
	})

	older := &nostr.Event{
		ID:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		PubKey:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Kind:      30443,
		CreatedAt: nostr.Timestamp(100),
		Tags:      nostr.Tags{{"d", "slot-1"}},
	}
	newer := &nostr.Event{
		ID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PubKey:    older.PubKey,
		Kind:      30443,
		CreatedAt: nostr.Timestamp(200),
		Tags:      nostr.Tags{{"d", "slot-1"}},
	}

	var gotFilter nostr.Filter
	var deleteCalls int
	var deletedID string
	var deletedBy string

	queryFirstStoredEvent = func(_ context.Context, filter nostr.Filter) (*nostr.Event, error) {
		gotFilter = filter
		return older, nil
	}
	deleteStoredEventByID = func(_ context.Context, id, replacedBy string) error {
		deleteCalls++
		deletedID = id
		deletedBy = replacedBy
		return nil
	}

	if err := prepareEventForStorage(context.Background(), newer); err != nil {
		t.Fatalf("prepareEventForStorage() error = %v", err)
	}

	if len(gotFilter.Authors) != 1 || gotFilter.Authors[0] != newer.PubKey {
		t.Fatalf("prepareEventForStorage() authors filter = %#v", gotFilter.Authors)
	}
	if len(gotFilter.Kinds) != 1 || gotFilter.Kinds[0] != 30443 {
		t.Fatalf("prepareEventForStorage() kinds filter = %#v", gotFilter.Kinds)
	}
	dValues := gotFilter.Tags["d"]
	if len(dValues) != 1 || dValues[0] != "slot-1" {
		t.Fatalf("prepareEventForStorage() d-tag filter = %#v", dValues)
	}
	if deleteCalls != 1 {
		t.Fatalf("deleteStoredEventByID call count = %d, want 1", deleteCalls)
	}
	if deletedID != older.ID {
		t.Fatalf("deleted event id = %q, want %q", deletedID, older.ID)
	}
	if deletedBy != newer.ID {
		t.Fatalf("deleted_by marker = %q, want %q", deletedBy, newer.ID)
	}
}
