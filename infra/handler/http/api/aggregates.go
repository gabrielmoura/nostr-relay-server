package api

import (
	"errors"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

type kindCount struct {
	Kind  int `json:"kind"`
	Count int `json:"count"`
}

type authorCount struct {
	Pubkey string `json:"pubkey"`
	Count  int    `json:"count"`
}

func (h *Handler) aggregateLimit(c *fiber.Ctx) (int, error) {
	maximum := 100
	if configured := h.maxLimit(); configured > 0 && configured < maximum {
		maximum = configured
	}
	defaultLimit := min(20, maximum)
	limit, err := positiveInt(c.Query("limit"), defaultLimit)
	if err != nil || limit > maximum {
		return 0, errors.New("limit must be a positive integer within the configured maximum")
	}
	return limit, nil
}

func (h *Handler) relayKinds(c *fiber.Ctx) error {
	limit, err := h.aggregateLimit(c)
	if err != nil {
		return badRequest(c, err)
	}
	events, exhausted, err := h.scan(c.UserContext(), nostr.Filter{}, 0, h.maxScan(), excludedTags(c), false)
	if err != nil {
		return internalError(c, err)
	}
	kinds := countKinds(events)
	sort.Slice(kinds, func(i, j int) bool {
		if kinds[i].Count != kinds[j].Count {
			return kinds[i].Count > kinds[j].Count
		}
		return kinds[i].Kind < kinds[j].Kind
	})
	if len(kinds) > limit {
		kinds = kinds[:limit]
	}
	return c.JSON(fiber.Map{"kinds": kinds, "approximate": !exhausted, "filtered": true})
}

func (h *Handler) relayTopAuthors(c *fiber.Ctx) error {
	limit, err := h.aggregateLimit(c)
	if err != nil {
		return badRequest(c, err)
	}
	events, exhausted, err := h.scan(c.UserContext(), nostr.Filter{}, 0, h.maxScan(), excludedTags(c), false)
	if err != nil {
		return internalError(c, err)
	}
	counts := make(map[string]int)
	for _, event := range events {
		counts[event.PubKey]++
	}
	authors := make([]authorCount, 0, len(counts))
	for pubkey, count := range counts {
		authors = append(authors, authorCount{Pubkey: pubkey, Count: count})
	}
	sort.Slice(authors, func(i, j int) bool {
		if authors[i].Count != authors[j].Count {
			return authors[i].Count > authors[j].Count
		}
		return authors[i].Pubkey < authors[j].Pubkey
	})
	if len(authors) > limit {
		authors = authors[:limit]
	}
	return c.JSON(fiber.Map{"authors": authors, "approximate": !exhausted, "filtered": true})
}

func (h *Handler) authorAggregate(c *fiber.Ctx) ([]*nostr.Event, bool, nostr.Filter, error) {
	pubkey, err := decodeAuthor(c.Params("npub"))
	if err != nil {
		return nil, false, nostr.Filter{}, err
	}
	filter := nostr.Filter{Authors: []string{pubkey}}
	events, exhausted, err := h.scan(c.UserContext(), filter, 0, h.maxScan(), excludedTags(c), false)
	return events, exhausted, filter, err
}

func (h *Handler) authorKinds(c *fiber.Ctx) error {
	if _, err := decodeAuthor(c.Params("npub")); err != nil {
		return badRequest(c, err)
	}
	events, exhausted, _, err := h.authorAggregate(c)
	if err != nil {
		return internalError(c, err)
	}
	kinds := countKinds(events)
	sort.Slice(kinds, func(i, j int) bool {
		if kinds[i].Count != kinds[j].Count {
			return kinds[i].Count > kinds[j].Count
		}
		return kinds[i].Kind < kinds[j].Kind
	})
	return c.JSON(fiber.Map{"kinds": kinds, "approximate": !exhausted})
}

func (h *Handler) authorStats(c *fiber.Ctx) error {
	if _, err := decodeAuthor(c.Params("npub")); err != nil {
		return badRequest(c, err)
	}
	events, exhausted, filter, err := h.authorAggregate(c)
	if err != nil {
		return internalError(c, err)
	}
	first, err := h.pageWithOrder(c.UserContext(), withLimit(filter, 1), 0, excludedTags(c), true)
	if err != nil {
		return internalError(c, err)
	}
	last, err := h.pageWithOrder(c.UserContext(), withLimit(filter, 1), 0, excludedTags(c), false)
	if err != nil {
		return internalError(c, err)
	}
	var firstSeen, lastSeen *int64
	var firstMonth, lastMonth *string
	if len(first.Events) > 0 {
		v := int64(first.Events[0].CreatedAt)
		month := time.Unix(v, 0).UTC().Format("2006-01")
		firstSeen, firstMonth = &v, &month
	}
	if len(last.Events) > 0 {
		v := int64(last.Events[0].CreatedAt)
		month := time.Unix(v, 0).UTC().Format("2006-01")
		lastSeen, lastMonth = &v, &month
	}
	return c.JSON(fiber.Map{"total": len(events), "approximate": !exhausted, "first_seen": firstSeen, "last_seen": lastSeen, "first_month": firstMonth, "last_month": lastMonth, "kinds": countKinds(events)})
}

func withLimit(filter nostr.Filter, limit int) nostr.Filter { filter.Limit = limit; return filter }

func countKinds(events []*nostr.Event) []kindCount {
	counts := make(map[int]int)
	for _, event := range events {
		counts[event.Kind]++
	}
	kinds := make([]kindCount, 0, len(counts))
	for kind, count := range counts {
		kinds = append(kinds, kindCount{Kind: kind, Count: count})
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i].Kind < kinds[j].Kind })
	return kinds
}
