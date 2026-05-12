package http

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func buildAdminFilter(c *fiber.Ctx) nostr.Filter {
	queryArgs := c.Request().URI().QueryArgs()
	tags := make(nostr.TagMap)
	for _, raw := range queryArgs.PeekMulti("tag") {
		part := string(raw)
		pieces := strings.SplitN(part, ":", 2)
		if len(pieces) != 2 {
			continue
		}
		key := strings.TrimPrefix(pieces[0], "#")
		if normalized, err := normalizeTagQueryValue(key, pieces[1]); err == nil {
			tags[key] = append(tags[key], normalized)
			continue
		}
		tags[key] = append(tags[key], pieces[1])
	}

	if rawTags := strings.TrimSpace(c.Query("tags")); rawTags != "" {
		for _, tag := range strings.Split(rawTags, ",") {
			normalized := strings.TrimSpace(strings.TrimPrefix(tag, "#"))
			if normalized == "" {
				continue
			}
			tags["t"] = append(tags["t"], normalized)
		}
	}

	authors := queryValues(queryArgs.PeekMulti("author"))
	normalizedAuthors := make([]string, 0, len(authors))
	for _, author := range authors {
		if normalized, err := normalizePublicKey(author); err == nil {
			normalizedAuthors = append(normalizedAuthors, normalized)
		}
	}

	search := c.Query("q")
	if normalized, err := normalizeSearchQuery(search); err == nil {
		search = normalized
	}
	if eventIDPattern.MatchString(search) {
		return nostr.Filter{IDs: []string{search}, Authors: normalizedAuthors, Kinds: parseKinds(queryArgs.PeekMulti("kind")), Tags: tags, Limit: adminLimit(c)}
	}
	if publicKeyPattern.MatchString(search) {
		normalizedAuthors = append(normalizedAuthors, search)
		search = ""
	}

	return nostr.Filter{
		Search:  search,
		Authors: normalizedAuthors,
		Kinds:   parseKinds(queryArgs.PeekMulti("kind")),
		Tags:    tags,
		Limit:   adminLimit(c),
	}
}

func queryValues(values [][]byte) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func parseKinds(values [][]byte) []int {
	result := make([]int, 0, len(values))
	for _, value := range values {
		kind, err := strconv.Atoi(string(value))
		if err == nil {
			result = append(result, kind)
		}
	}
	return result
}

func normalizeRelayURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "wss", "ws":
		return parsed.String(), true
	case "https":
		parsed.Scheme = "wss"
		return parsed.String(), true
	case "http":
		parsed.Scheme = "ws"
		return parsed.String(), true
	default:
		return "", false
	}
}

func mergeFetchRelayList(groups ...[]string) []string {
	seen := make(map[string]struct{}, 16)
	merged := make([]string, 0, 16)
	for _, relays := range groups {
		for _, relay := range relays {
			normalized, ok := normalizeRelayURL(relay)
			if !ok {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			merged = append(merged, normalized)
		}
	}
	return merged
}

func normalizePublicKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing public key")
	}
	if strings.HasPrefix(value, "npub") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil || prefix != "npub" {
			return "", fmt.Errorf("invalid npub: %w", err)
		}
		pubkey, ok := decoded.(string)
		if !ok {
			return "", fmt.Errorf("invalid npub payload")
		}
		return pubkey, nil
	}
	if strings.HasPrefix(value, "nprofile") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil || prefix != "nprofile" {
			return "", fmt.Errorf("invalid nprofile: %w", err)
		}
		profile, ok := decoded.(nostr.ProfilePointer)
		if !ok || profile.PublicKey == "" {
			return "", fmt.Errorf("invalid nprofile payload")
		}
		return profile.PublicKey, nil
	}
	return value, nil
}

func normalizeEventID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing event id")
	}
	if strings.HasPrefix(value, "note") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil || prefix != "note" {
			return "", fmt.Errorf("invalid note: %w", err)
		}
		id, ok := decoded.(string)
		if !ok || id == "" {
			return "", fmt.Errorf("invalid note payload")
		}
		return id, nil
	}
	if strings.HasPrefix(value, "nevent") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil || prefix != "nevent" {
			return "", fmt.Errorf("invalid nevent: %w", err)
		}
		eventPtr, ok := decoded.(nostr.EventPointer)
		if !ok || eventPtr.ID == "" {
			return "", fmt.Errorf("invalid nevent payload")
		}
		return eventPtr.ID, nil
	}
	return value, nil
}

func normalizeAddressValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing address value")
	}
	if strings.HasPrefix(value, "naddr") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil || prefix != "naddr" {
			return "", fmt.Errorf("invalid naddr: %w", err)
		}
		entityPtr, ok := decoded.(nostr.EntityPointer)
		if !ok || entityPtr.PublicKey == "" {
			return "", fmt.Errorf("invalid naddr payload")
		}
		return fmt.Sprintf("%d:%s:%s", entityPtr.Kind, entityPtr.PublicKey, entityPtr.Identifier), nil
	}
	return value, nil
}

func normalizeSearchQuery(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "npub") || strings.HasPrefix(value, "nprofile") {
		return normalizePublicKey(value)
	}
	if strings.HasPrefix(value, "note") || strings.HasPrefix(value, "nevent") {
		return normalizeEventID(value)
	}
	return value, nil
}

func normalizeTagQueryValue(key string, value string) (string, error) {
	switch strings.TrimSpace(strings.TrimPrefix(key, "#")) {
	case "e", "q":
		return normalizeEventID(value)
	case "p":
		return normalizePublicKey(value)
	case "a":
		return normalizeAddressValue(value)
	default:
		return value, nil
	}
}
