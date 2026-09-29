package api

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func parseFilter(c *fiber.Ctx, maxLimit int) (nostr.Filter, int, error) {
	filter := nostr.Filter{Tags: make(nostr.TagMap)}
	args := c.Request().URI().QueryArgs()
	filter.IDs = stringValues(args.PeekMulti("id"))
	filter.Authors = stringValues(args.PeekMulti("author"))
	filter.Kinds = intValues(args.PeekMulti("kind"))
	filter.Search = strings.TrimSpace(c.Query("search"))
	for _, key := range []string{"e", "p", "t", "d"} {
		values := stringValues(args.PeekMulti(key))
		if len(values) > 0 {
			filter.Tags[key] = values
		}
	}
	if len(filter.Tags) == 0 {
		filter.Tags = nil
	}
	if filter.Search == "" {
		filter.Search = strings.TrimSpace(c.Query("q"))
	}
	if err := normalizeFilterIDs(&filter); err != nil {
		return nostr.Filter{}, 0, err
	}

	limit, err := positiveInt(c.Query("limit"), 100)
	if err != nil || limit > maxLimit {
		return nostr.Filter{}, 0, errors.New("limit must be a positive integer within the configured maximum")
	}
	filter.Limit = limit
	offset, err := nonNegativeInt(c.Query("offset"), 0)
	if err != nil {
		return nostr.Filter{}, 0, errors.New("offset must be a non-negative integer")
	}
	if filter.Since, err = timestamp(c.Query("since")); err != nil {
		return nostr.Filter{}, 0, errors.New("since must be a Unix timestamp")
	}
	if filter.Until, err = timestamp(c.Query("until")); err != nil {
		return nostr.Filter{}, 0, errors.New("until must be a Unix timestamp")
	}
	if filter.Since != nil && filter.Until != nil && *filter.Since > *filter.Until {
		return nostr.Filter{}, 0, errors.New("since must not be greater than until")
	}
	return filter, offset, nil
}

func excludedTags(c *fiber.Ctx) map[string]map[string]struct{} {
	args := c.Request().URI().QueryArgs()
	excluded := make(map[string]map[string]struct{})
	for _, key := range []string{"e", "p", "t", "d"} {
		values := stringValues(args.PeekMulti("no_" + key))
		if len(values) == 0 {
			continue
		}
		set := make(map[string]struct{}, len(values))
		for _, value := range values {
			set[value] = struct{}{}
		}
		excluded[key] = set
	}
	return excluded
}

func normalizeFilterIDs(filter *nostr.Filter) error {
	for index, value := range filter.IDs {
		if !isHexID(value) {
			return errors.New("id must be a 64-character hexadecimal event id")
		}
		filter.IDs[index] = strings.ToLower(value)
	}
	for index, value := range filter.Authors {
		if !isHexID(value) {
			return errors.New("author must be a 64-character hexadecimal public key")
		}
		filter.Authors[index] = strings.ToLower(value)
	}
	for key, values := range filter.Tags {
		for index, value := range values {
			if key == "e" {
				id, err := decodeEventIDOrHex(value)
				if err != nil {
					return err
				}
				filter.Tags[key][index] = id
			}
			if key == "p" {
				pubkey, err := decodeNPubOrHex(value)
				if err != nil {
					return err
				}
				filter.Tags[key][index] = pubkey
			}
		}
	}
	return nil
}

func decodeEventIDOrHex(value string) (string, error) {
	if isHexID(value) {
		return strings.ToLower(value), nil
	}
	return decodeEventID(value)
}

func decodeNPubOrHex(value string) (string, error) {
	if isHexID(value) {
		return strings.ToLower(value), nil
	}
	return decodeNPub(value)
}

func stringValues(values [][]byte) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(string(value))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func intValues(values [][]byte) []int {
	result := make([]int, 0, len(values))
	for _, value := range values {
		parsed, err := strconv.Atoi(string(value))
		if err == nil && parsed >= 0 {
			result = append(result, parsed)
		}
	}
	return result
}

func positiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("invalid positive integer")
	}
	return value, nil
}

func nonNegativeInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, errors.New("invalid non-negative integer")
	}
	return value, nil
}

func timestamp(raw string) (*nostr.Timestamp, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return nil, errors.New("invalid timestamp")
	}
	timestamp := nostr.Timestamp(value)
	return &timestamp, nil
}
