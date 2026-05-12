package db

import (
	"sort"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

func scanAdminLabelEvent(scanner interface{ Scan(dest ...any) error }) (*nostr.Event, error) {
	return scanNostrEvent(scanner)
}

func mapAdminLabelRecord(evt *nostr.Event) AdminLabelRecord {
	return AdminLabelRecord{
		Event:     evt,
		Namespace: adminLabelNamespace(evt),
		Labels:    adminLabelValues(evt),
		Target:    adminLabelTarget(evt),
	}
}

func adminLabelNamespace(evt *nostr.Event) string {
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "L" {
			return tag[1]
		}
	}
	return "ugc"
}

func adminLabelValues(evt *nostr.Event) []string {
	labels := make([]string, 0, 2)
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "l" {
			labels = append(labels, tag[1])
		}
	}
	return labels
}

func adminLabelTarget(evt *nostr.Event) AdminLabelTarget {
	for _, candidate := range []struct {
		tag      string
		typeName string
	}{{"e", "event"}, {"p", "pubkey"}, {"a", "address"}, {"r", "reference"}, {"t", "topic"}} {
		for _, tag := range evt.Tags {
			if len(tag) >= 2 && tag[0] == candidate.tag {
				target := AdminLabelTarget{Type: candidate.typeName, Value: tag[1]}
				if (candidate.tag == "e" || candidate.tag == "p") && len(tag) >= 3 {
					target.RelayHint = tag[2]
				}
				return target
			}
		}
	}
	return AdminLabelTarget{}
}

func sortAdminLabelCounts(counts map[string]int64) []AdminLabelCount {
	items := make([]AdminLabelCount, 0, len(counts))
	for key, count := range counts {
		if strings.TrimSpace(key) == "" {
			continue
		}
		items = append(items, AdminLabelCount{Key: key, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	return items
}
