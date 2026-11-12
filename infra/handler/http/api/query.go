package api

import (
	"context"
	"errors"

	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	"github.com/nbd-wtf/go-nostr"
)

var errScanLimit = errors.New("visible scan limit exceeded")

func (h *Handler) page(ctx context.Context, filter nostr.Filter, offset int) (eventsResponse, error) {
	return h.pageWithExcluded(ctx, filter, offset, nil)
}

func (h *Handler) pageWithExcluded(ctx context.Context, filter nostr.Filter, offset int, excluded map[string]map[string]struct{}) (eventsResponse, error) {
	return h.pageWithOrder(ctx, filter, offset, excluded, false)
}

func (h *Handler) pageWithOrder(ctx context.Context, filter nostr.Filter, offset int, excluded map[string]map[string]struct{}, ascending bool) (eventsResponse, error) {
	visible, exhausted, err := h.scan(ctx, filter, offset, filter.Limit+1, excluded, ascending)
	if err != nil {
		return eventsResponse{}, err
	}
	more := len(visible) > filter.Limit
	if !more && !exhausted {
		return eventsResponse{}, errScanLimit
	}
	if more {
		visible = visible[:filter.Limit]
	}
	return eventsResponse{Events: visible, Count: len(visible), More: more}, nil
}

func (h *Handler) visibleCount(ctx context.Context, filter nostr.Filter) (int, error) {
	return h.visibleCountWithExcluded(ctx, filter, nil)
}

func (h *Handler) visibleCountWithExcluded(ctx context.Context, filter nostr.Filter, excluded map[string]map[string]struct{}) (int, error) {
	events, exhausted, err := h.scan(ctx, filter, 0, h.maxScan(), excluded, false)
	if err != nil {
		return 0, err
	}
	if !exhausted {
		return 0, errScanLimit
	}
	return len(events), nil
}

func (h *Handler) scan(ctx context.Context, filter nostr.Filter, visibleOffset, wanted int, excluded map[string]map[string]struct{}, ascending bool) ([]*nostr.Event, bool, error) {
	maxScan := h.maxScan()
	batchSize := h.maxLimit()
	if batchSize < wanted {
		batchSize = wanted
	}
	if batchSize < 1 {
		batchSize = 1
	}
	if h.cfg.Relay.QueryLimit > 0 && batchSize > h.cfg.Relay.QueryLimit {
		batchSize = h.cfg.Relay.QueryLimit
	}
	result := make([]*nostr.Event, 0, wanted)
	rawOffset := 0
	scanned := 0
	skipped := 0
	for scanned < maxScan {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		remaining := maxScan - scanned
		batch := batchSize
		if batch > remaining {
			batch = remaining
		}
		candidateFilter := filter
		candidateFilter.Limit = batch
		var candidates []*nostr.Event
		var err error
		if h.orderedSource != nil {
			candidates, err = h.orderedSource(ctx, candidateFilter, rawOffset, ascending)
		} else {
			candidates, err = h.source(ctx, candidateFilter, rawOffset)
		}
		if err != nil {
			return nil, false, err
		}
		scanned += len(candidates)
		rawOffset += len(candidates)
		visible, err := visibleEvents(ctx, candidateFilter, candidates, excluded)
		if err != nil {
			return nil, false, err
		}
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		for _, event := range visible {
			if skipped < visibleOffset {
				skipped++
				continue
			}
			result = append(result, event)
			if len(result) >= wanted {
				return result, false, nil
			}
		}
		if len(candidates) < batch {
			return result, true, nil
		}
	}
	return result, false, nil
}

func (h *Handler) maxScan() int {
	if h.cfg.API.MaxScan > 0 {
		return h.cfg.API.MaxScan
	}
	return 2000
}

func visibleEvents(ctx context.Context, filter nostr.Filter, candidates []*nostr.Event, excluded map[string]map[string]struct{}) ([]*nostr.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	public := make([]*nostr.Event, 0, len(candidates))
	for _, event := range candidates {
		if event != nil && event.Kind != 1059 && event.Kind != nostr.KindEncryptedDirectMessage && !hasProtectedTag(event) && !hasExcludedTag(event, excluded) {
			public = append(public, event)
		}
	}
	return groups.FilterEvents(ctx, "", filter, public)
}

func hasExcludedTag(event *nostr.Event, excluded map[string]map[string]struct{}) bool {
	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		if values, ok := excluded[tag[0]]; ok {
			if _, found := values[tag[1]]; found {
				return true
			}
		}
	}
	return false
}

func hasProtectedTag(event *nostr.Event) bool {
	for _, tag := range event.Tags {
		if len(tag) > 0 && tag[0] == "-" {
			return true
		}
	}
	return false
}
