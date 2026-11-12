package listener

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
)

func TestNotifyLocalListenersHonorsNIP29Visibility(t *testing.T) {
	restore := installLiveListenerTestState(t)
	defer restore()

	SetLiveEventAuthorizer(func(_ context.Context, authed string, _ *nostr.Event) (bool, error) {
		return authed == "member", nil
	})

	anonymous := testWS("")
	member := testWS("member")
	nonMember := testWS("non-member")
	installSubscription(anonymous, "anonymous")
	installSubscription(member, "member")
	installSubscription(nonMember, "non-member")

	notifyLocalListeners(&nostr.Event{ID: "group-event", Kind: nostr.KindTextNote})

	assertReceivesEvent(t, member.ChanSender, "member")
	assertNoEvent(t, anonymous.ChanSender, "anonymous")
	assertNoEvent(t, nonMember.ChanSender, "non-member")
}

func TestNotifyLocalListenersFailsClosedWhenNIP29AuthorizationErrors(t *testing.T) {
	restore := installLiveListenerTestState(t)
	defer restore()

	SetLiveEventAuthorizer(func(context.Context, string, *nostr.Event) (bool, error) {
		return false, errors.New("group projection unavailable")
	})

	ws := testWS("member")
	installSubscription(ws, "member")
	notifyLocalListeners(&nostr.Event{ID: "group-event", Kind: nostr.KindTextNote})

	assertNoEvent(t, ws.ChanSender, "authorization error")
}

func TestNotifyLocalListenersMemoizesAuthorizationPerIdentityAfterFilterMatch(t *testing.T) {
	restore := installLiveListenerTestState(t)
	defer restore()

	lookups := make(map[string]int)
	SetLiveEventAuthorizer(func(_ context.Context, authed string, _ *nostr.Event) (bool, error) {
		lookups[authed]++
		return true, nil
	})

	firstMember := testWS("member")
	secondMember := testWS("member")
	noMatch := testWS("no-match")
	installSubscription(firstMember, "first-member")
	installSubscription(secondMember, "second-member")
	installSubscriptionWithFilters(noMatch, "no-match", nostr.Filters{{Kinds: []int{nostr.KindReaction}}})

	notifyLocalListeners(&nostr.Event{ID: "group-event", Kind: nostr.KindTextNote})

	if got := lookups["member"]; got != 1 {
		t.Fatalf("member authorization lookups = %d, want 1", got)
	}
	if got := lookups["no-match"]; got != 0 {
		t.Fatalf("non-matching subscription authorization lookups = %d, want 0", got)
	}
	assertReceivesEvent(t, firstMember.ChanSender, "first-member")
	assertReceivesEvent(t, secondMember.ChanSender, "second-member")
	assertNoEvent(t, noMatch.ChanSender, "no-match")
}

func installLiveListenerTestState(t *testing.T) func() {
	t.Helper()
	liveEventAuthorizer.RLock()
	previousAuthorizer := liveEventAuthorizer.fn
	liveEventAuthorizer.RUnlock()
	previousConfig := config.Cfg
	config.Cfg = &config.Config{}

	localListenersMutex.Lock()
	previousListeners := localListeners
	localListeners = make(map[*dto.WsServer]map[string]*Listener)
	localListenersMutex.Unlock()

	return func() {
		localListenersMutex.Lock()
		localListeners = previousListeners
		localListenersMutex.Unlock()
		SetLiveEventAuthorizer(previousAuthorizer)
		config.Cfg = previousConfig
	}
}

func testWS(authed string) *dto.WsServer {
	return &dto.WsServer{
		Authed:     authed,
		Ctx:        context.Background(),
		ChanSender: make(chan any, 1),
	}
}

func installSubscription(ws *dto.WsServer, id string) {
	installSubscriptionWithFilters(ws, id, nostr.Filters{{Kinds: []int{nostr.KindTextNote}}})
}

func installSubscriptionWithFilters(ws *dto.WsServer, id string, filters nostr.Filters) {
	localListenersMutex.Lock()
	defer localListenersMutex.Unlock()
	localListeners[ws] = map[string]*Listener{
		id: {filters: filters},
	}
}

func assertReceivesEvent(t *testing.T, messages <-chan any, wantSubscription string) {
	t.Helper()
	select {
	case message := <-messages:
		envelope, ok := message.(nostr.EventEnvelope)
		if !ok || envelope.SubscriptionID == nil || *envelope.SubscriptionID != wantSubscription {
			t.Fatalf("unexpected live delivery: %#v", message)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected live delivery for %q", wantSubscription)
	}
}

func assertNoEvent(t *testing.T, messages <-chan any, recipient string) {
	t.Helper()
	select {
	case message := <-messages:
		t.Fatalf("unexpected live delivery for %q: %#v", recipient, message)
	default:
	}
}
