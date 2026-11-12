package event

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	nostr_custom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

const discoveryDefaultResults = 20
const discoveryMaxResults = 100

type discoveryRequest struct {
	pubKey string
	limit  int
}

var discoveryJobs = struct {
	sync.Mutex
	inFlight map[string]struct{}
}{inFlight: make(map[string]struct{})}

var discoverySlots = make(chan struct{}, 4)

// reserveDiscovery acquires capacity before acknowledging a request so a busy
// service does not persist a request that cannot be retried.
func reserveDiscovery(evt *nostr.Event) bool {
	discoveryJobs.Lock()
	if _, exists := discoveryJobs.inFlight[evt.ID]; exists {
		discoveryJobs.Unlock()
		return false
	}
	discoveryJobs.inFlight[evt.ID] = struct{}{}
	discoveryJobs.Unlock()

	select {
	case discoverySlots <- struct{}{}:
	default:
		discoveryJobs.Lock()
		delete(discoveryJobs.inFlight, evt.ID)
		discoveryJobs.Unlock()
		return false
	}
	return true
}

func releaseDiscovery(id string) {
	<-discoverySlots
	discoveryJobs.Lock()
	delete(discoveryJobs.inFlight, id)
	discoveryJobs.Unlock()
}

func startDiscovery(ctx context.Context, evt *nostr.Event) {
	go func() {
		defer releaseDiscovery(evt.ID)
		handleContentDiscovery(ctx, evt)
	}()
}

func handleContentDiscovery(parent context.Context, evt *nostr.Event) {
	// The request is already durable; closing the originating connection must
	// not cancel the public response. The timeout still bounds the work.
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	completed, err := discoveryResultExists(ctx, evt.ID)
	if err != nil {
		sendDiscoveryFeedback(evt, err)
		return
	}
	if completed {
		return
	}
	ids, err := handleJobContentDiscovery(ctx, evt)
	if err != nil {
		sendDiscoveryFeedback(evt, err)
		return
	}
	resp, err := contentDiscoveryResponse(evt, ids)
	if err != nil {
		sendDiscoveryFeedback(evt, err)
		return
	}
	if _, accepted := ingestion.PushDurable(ctx, &resp); !accepted {
		sendDiscoveryFeedback(evt, fmt.Errorf("response ingestion queue is full"))
	}
}

func discoveryResultExists(ctx context.Context, requestID string) (bool, error) {
	if internaldb.DbQueries == nil {
		return false, fmt.Errorf("event store is unavailable")
	}
	events, err := internaldb.DbQueries.QueryEvents(ctx, nostr.Filter{
		Authors: []string{config.Cfg.RelayInformation.PubKey}, Kinds: []int{nostr_custom.KindContentDiscoveryResponse},
		Tags: nostr.TagMap{"e": []string{requestID}}, Limit: 1,
	})
	if err != nil {
		return false, fmt.Errorf("query discovery response: %w", err)
	}
	return len(events) > 0, nil
}

func handleJobContentDiscovery(ctx context.Context, evt *nostr.Event) ([]string, error) {
	req, err := parseDiscoveryRequest(evt)
	if err != nil {
		return nil, err
	}
	queries := internaldb.DbQueries
	if queries == nil {
		return nil, fmt.Errorf("event store is unavailable")
	}
	follows, err := queryDiscoveryEvents(ctx, queries, nostr.Filter{Authors: []string{req.pubKey}, Kinds: []int{3}, Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("query contact list: %w", err)
	}
	if len(follows) == 0 {
		return []string{}, nil
	}
	if hasTag(follows[0].Tags, "-") || hasTag(follows[0].Tags, "h") {
		return []string{}, nil
	}
	authors := followedPubkeys(follows[0].Tags)
	if len(authors) == 0 {
		return []string{}, nil
	}
	if max := config.Cfg.Relay.QueryAuthorsLimit; max > 0 && len(authors) > max {
		authors = authors[:max]
	}
	events, err := queryDiscoveryEvents(ctx, queries, nostr.Filter{Authors: authors, Kinds: []int{nostr.KindTextNote}, Limit: req.limit})
	if err != nil {
		return nil, fmt.Errorf("query followed notes: %w", err)
	}
	ids := make([]string, 0, len(events))
	for _, event := range events {
		if event != nil && !hasTag(event.Tags, "-") && !hasTag(event.Tags, "h") {
			ids = append(ids, event.ID)
		}
	}
	return ids, nil
}

func hasDiscoveryEncryption(tags nostr.Tags) bool {
	for _, tag := range tags {
		if len(tag) > 0 && tag[0] == "encrypted" {
			return true
		}
		if len(tag) > 1 && tag[0] == "param" && tag[1] == "encrypted" {
			return true
		}
	}
	return false
}

// Discovery responses are public relay events. Apply NIP-29 as an anonymous
// reader even when the request names a group member, so a public 6300 can
// never disclose IDs from a private group.
func queryDiscoveryEvents(ctx context.Context, queries *db.Queries, filter nostr.Filter) ([]*nostr.Event, error) {
	events, err := queries.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	return groups.FilterEvents(ctx, "", filter, events)
}

func hasTag(tags nostr.Tags, name string) bool {
	for _, tag := range tags {
		if len(tag) > 0 && tag[0] == name {
			return true
		}
	}
	return false
}

func parseDiscoveryRequest(evt *nostr.Event) (discoveryRequest, error) {
	req := discoveryRequest{pubKey: evt.PubKey, limit: discoveryDefaultResults}
	seen := map[string]string{}
	target := ""
	for _, tag := range evt.Tags {
		if len(tag) < 3 || tag[0] != "param" {
			continue
		}
		name, value := tag[1], tag[2]
		if previous, ok := seen[name]; ok && previous != value {
			return discoveryRequest{}, fmt.Errorf("conflicting param %q", name)
		}
		seen[name] = value
		switch name {
		case "p", "user":
			if target != "" && target != value {
				return discoveryRequest{}, fmt.Errorf("conflicting target pubkeys")
			}
			target = value
			req.pubKey = value
		case "max_results":
			limit, err := strconv.Atoi(value)
			if err != nil || limit < 1 || limit > discoveryMaxResults {
				return discoveryRequest{}, fmt.Errorf("max_results must be between 1 and %d", discoveryMaxResults)
			}
			req.limit = limit
		}
	}
	if decoded, err := hex.DecodeString(req.pubKey); err != nil || len(decoded) != 32 {
		return discoveryRequest{}, fmt.Errorf("p must be a 32-byte hexadecimal pubkey")
	}
	return req, nil
}

func followedPubkeys(tags nostr.Tags) []string {
	seen := make(map[string]struct{})
	pubkeys := make([]string, 0)
	for _, tag := range tags {
		if len(tag) < 2 || tag[0] != "p" || len(tag[1]) != 64 {
			continue
		}
		if _, err := hex.DecodeString(tag[1]); err != nil {
			continue
		}
		if _, ok := seen[tag[1]]; !ok {
			seen[tag[1]] = struct{}{}
			pubkeys = append(pubkeys, tag[1])
		}
	}
	return pubkeys
}

func contentDiscoveryResponse(request *nostr.Event, ids []string) (nostr.Event, error) {
	content := make(nostr.Tags, 0, len(ids))
	for _, id := range ids {
		content = append(content, nostr.Tag{"e", id})
	}
	payload, err := json.Marshal(content)
	if err != nil {
		return nostr.Event{}, fmt.Errorf("encode discovery response: %w", err)
	}
	response := nostr.Event{Kind: nostr_custom.KindContentDiscoveryResponse, PubKey: config.Cfg.RelayInformation.PubKey, CreatedAt: nostr.Now(), Content: string(payload), Tags: nostr.Tags{{"request", request.String()}, {"e", request.ID}, {"p", request.PubKey}}}
	for _, tag := range request.Tags {
		if len(tag) > 0 && tag[0] == "i" {
			response.Tags = append(response.Tags, append(nostr.Tag(nil), tag...))
		}
	}
	if strings.TrimSpace(config.Cfg.RelayInformation.PrivKey) == "" {
		return nostr.Event{}, fmt.Errorf("relay signing key is unavailable")
	}
	if err := response.Sign(config.Cfg.RelayInformation.PrivKey); err != nil {
		return nostr.Event{}, fmt.Errorf("sign discovery response: %w", err)
	}
	return response, nil
}

func sendDiscoveryFeedback(request *nostr.Event, err error) {
	// Keep details in server logs, not in public feedback.
	log.Logger.Warn("content discovery failed", zap.Error(err), zap.String("request_id", request.ID))
	feedback := nostr.Event{Kind: 7000, PubKey: config.Cfg.RelayInformation.PubKey, CreatedAt: nostr.Now(), Content: "error: discovery unavailable", Tags: nostr.Tags{{"e", request.ID}, {"p", request.PubKey}, {"status", "error"}}}
	if signErr := feedback.Sign(config.Cfg.RelayInformation.PrivKey); signErr != nil {
		log.Logger.Error("failed to sign discovery feedback", zap.Error(signErr))
	} else if !ingestion.Push(&feedback) {
		log.Logger.Warn("failed to enqueue discovery feedback")
	}
}
