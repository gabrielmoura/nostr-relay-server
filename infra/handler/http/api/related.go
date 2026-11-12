package api

import (
	"errors"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func (h *Handler) relatedEvents(c *fiber.Ctx) error {
	id := c.Params("id")
	if !isHexID(id) {
		return badRequest(c, errors.New("id must be a 64-character hexadecimal event id"))
	}
	id = strings.ToLower(id)
	filter, offset, err := parseFilter(c, h.maxLimit())
	if err != nil {
		return badRequest(c, err)
	}
	ascending, err := parseSort(c.Query("sort"))
	if err != nil {
		return badRequest(c, err)
	}
	// Each sorted branch must provide enough visible events to establish the
	// requested merged page; refuse unbounded offsets before allocating.
	if offset >= h.maxScan() || filter.Limit >= h.maxScan()-offset {
		return internalError(c, errScanLimit)
	}
	wanted := offset + filter.Limit + 1
	copyTags := func() nostr.TagMap {
		tags := make(nostr.TagMap, len(filter.Tags)+1)
		for key, values := range filter.Tags {
			tags[key] = append([]string(nil), values...)
		}
		return tags
	}
	replies, quotes := filter, filter
	replies.Tags = copyTags()
	replies.Tags["e"] = append(replies.Tags["e"], id)
	quotes.Tags = copyTags()
	delete(quotes.Tags, "e")
	quotes.Tags["q"] = []string{id}
	seen := make(map[string]*nostr.Event)
	for _, branch := range []nostr.Filter{replies, quotes} {
		events, exhausted, err := h.scan(c.UserContext(), branch, 0, wanted, excludedTags(c), ascending)
		if err != nil {
			return internalError(c, err)
		}
		if !exhausted && len(events) < wanted {
			return internalError(c, errScanLimit)
		}
		for _, event := range events {
			seen[event.ID] = event
		}
	}
	events := make([]*nostr.Event, 0, len(seen))
	for _, event := range seen {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].CreatedAt != events[j].CreatedAt {
			if ascending {
				return events[i].CreatedAt < events[j].CreatedAt
			}
			return events[i].CreatedAt > events[j].CreatedAt
		}
		return events[i].ID < events[j].ID
	})
	if offset >= len(events) {
		return c.JSON(eventsResponse{Events: []*nostr.Event{}})
	}
	events = events[offset:]
	more := len(events) > filter.Limit
	if more {
		events = events[:filter.Limit]
	}
	return c.JSON(eventsResponse{Events: events, Count: len(events), More: more})
}
