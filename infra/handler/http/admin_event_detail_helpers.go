package http

import (
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func npubFromPublicKey(pubkey string) string {
	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil {
		return ""
	}
	return npub
}

func neventFromEventID(eventID string, pubkey string) string {
	nevent, err := nip19.EncodeEvent(eventID, []string{}, pubkey)
	if err != nil {
		return ""
	}
	return nevent
}

func extractHashtags(event *nostr.Event) []string {
	result := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	for _, tag := range event.Tags {
		if len(tag) <= 1 || tag[0] != "t" {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(tag[1]))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func extractImageURLs(event *nostr.Event) []string {
	matches := imageURLPattern.FindAllString(event.Content, -1)
	if len(matches) == 0 {
		return []string{}
	}
	unique := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, raw := range matches {
		if _, ok := unique[raw]; ok {
			continue
		}
		unique[raw] = struct{}{}
		result = append(result, raw)
	}
	return result
}

func extractReportCore(event *nostr.Event) (targetEventID string, targetPubkey string, reportType string) {
	for _, tag := range event.Tags {
		if len(tag) <= 1 {
			continue
		}
		switch tag[0] {
		case "e":
			if targetEventID == "" {
				targetEventID = tag[1]
			}
			if len(tag) > 2 && reportType == "" {
				reportType = tag[2]
			}
		case "p":
			if targetPubkey == "" {
				targetPubkey = tag[1]
			}
			if len(tag) > 2 && reportType == "" {
				reportType = tag[2]
			}
		case "x":
			if len(tag) > 2 && reportType == "" {
				reportType = tag[2]
			}
		}
	}
	return targetEventID, targetPubkey, reportType
}
