package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type AdminLabelFilters struct {
	Namespace  string
	Label      string
	TargetType string
	Target     string
	Author     string
	Query      string
}

type AdminLabelTarget struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	RelayHint string `json:"relay_hint,omitempty"`
}

type AdminLabelRecord struct {
	Event     *nostr.Event     `json:"event"`
	Namespace string           `json:"namespace"`
	Labels    []string         `json:"labels"`
	Target    AdminLabelTarget `json:"target"`
}

type AdminLabelCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type AdminLabelsSummary struct {
	TotalEvents  int64             `json:"total_events"`
	TotalTargets int64             `json:"total_targets"`
	Namespaces   []AdminLabelCount `json:"namespaces"`
	Labels       []AdminLabelCount `json:"labels"`
	TargetTypes  []AdminLabelCount `json:"target_types"`
}

func (q *Queries) GetLabels(
	ctx context.Context,
	filters AdminLabelFilters,
	limit int,
	offset int,
) ([]AdminLabelRecord, int64, error) {
	whereSQL, args := buildAdminLabelsWhere(filters)

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM event e WHERE %s", whereSQL)
	var total int64
	if err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	listSQL := fmt.Sprintf(`
SELECT e.id, e.pubkey, e.created_at, e.kind, e.tags, e.content, e.sig
FROM event e
WHERE %s
ORDER BY e.created_at DESC, e.id DESC
LIMIT $%d OFFSET $%d`, whereSQL, len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminLabelRecord, 0, limit)
	for rows.Next() {
		evt, err := scanAdminLabelEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, mapAdminLabelRecord(evt))
	}

	return items, total, rows.Err()
}

func (q *Queries) GetLabelsSummary(ctx context.Context, filters AdminLabelFilters) (AdminLabelsSummary, error) {
	whereSQL, args := buildAdminLabelsWhere(filters)
	listSQL := fmt.Sprintf(`
SELECT e.id, e.pubkey, e.created_at, e.kind, e.tags, e.content, e.sig
FROM event e
WHERE %s
ORDER BY e.created_at DESC, e.id DESC`, whereSQL)

	rows, err := q.db.Query(ctx, listSQL, args...)
	if err != nil {
		return AdminLabelsSummary{}, err
	}
	defer rows.Close()

	namespaceCounts := make(map[string]int64)
	labelCounts := make(map[string]int64)
	targetTypeCounts := make(map[string]int64)
	uniqueTargets := make(map[string]struct{})

	var totalEvents int64
	for rows.Next() {
		evt, err := scanAdminLabelEvent(rows)
		if err != nil {
			return AdminLabelsSummary{}, err
		}

		record := mapAdminLabelRecord(evt)
		totalEvents++
		namespaceCounts[record.Namespace]++
		targetTypeCounts[record.Target.Type]++
		uniqueTargets[record.Target.Type+":"+record.Target.Value] = struct{}{}
		for _, label := range record.Labels {
			labelCounts[label]++
		}
	}
	if err := rows.Err(); err != nil {
		return AdminLabelsSummary{}, err
	}

	return AdminLabelsSummary{
		TotalEvents:  totalEvents,
		TotalTargets: int64(len(uniqueTargets)),
		Namespaces:   sortAdminLabelCounts(namespaceCounts),
		Labels:       sortAdminLabelCounts(labelCounts),
		TargetTypes:  sortAdminLabelCounts(targetTypeCounts),
	}, nil
}

func buildAdminLabelsWhere(filters AdminLabelFilters) (string, []any) {
	clauses := []string{"e.kind = 1985"}
	args := make([]any, 0, 8)

	if filters.Namespace != "" {
		args = append(args, filters.Namespace)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM jsonb_array_elements(e.tags) tag
			WHERE tag->>0 = 'L' AND tag->>1 = $%d
		)`, len(args)))
	}

	if filters.Label != "" {
		args = append(args, strings.ToLower(filters.Label))
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM jsonb_array_elements(e.tags) tag
			WHERE tag->>0 = 'l' AND lower(tag->>1) = $%d
		)`, len(args)))
	}

	if filters.Author != "" {
		args = append(args, filters.Author)
		clauses = append(clauses, fmt.Sprintf("e.pubkey = $%d", len(args)))
	}

	targetTag := adminLabelTargetTag(filters.TargetType)
	if targetTag != "" {
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM jsonb_array_elements(e.tags) tag
			WHERE tag->>0 = '%s'
		)`, targetTag))
	}

	if filters.Target != "" {
		args = append(args, filters.Target)
		if targetTag != "" {
			clauses = append(clauses, fmt.Sprintf(`EXISTS (
				SELECT 1 FROM jsonb_array_elements(e.tags) tag
				WHERE tag->>0 = '%s' AND tag->>1 = $%d
			)`, targetTag, len(args)))
		} else {
			clauses = append(clauses, fmt.Sprintf(`EXISTS (
				SELECT 1 FROM jsonb_array_elements(e.tags) tag
				WHERE tag->>0 IN ('e', 'p', 'a', 'r', 't') AND tag->>1 = $%d
			)`, len(args)))
		}
	}

	if filters.Query != "" {
		args = append(args, "%"+filters.Query+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf(`(
			e.content ILIKE $%d
			OR e.pubkey ILIKE $%d
			OR EXISTS (
				SELECT 1 FROM jsonb_array_elements(e.tags) tag
				WHERE tag->>0 IN ('e', 'p', 'a', 'r', 't') AND tag->>1 ILIKE $%d
			)
		)`, idx, idx, idx))
	}

	return strings.Join(clauses, " AND "), args
}

func adminLabelTargetTag(targetType string) string {
	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "event":
		return "e"
	case "pubkey":
		return "p"
	case "address":
		return "a"
	case "reference":
		return "r"
	case "topic":
		return "t"
	default:
		return ""
	}
}

func scanAdminLabelEvent(scanner interface{ Scan(dest ...any) error }) (*nostr.Event, error) {
	var evt nostr.Event
	var createdAt int64
	if err := scanner.Scan(
		&evt.ID,
		&evt.PubKey,
		&createdAt,
		&evt.Kind,
		&evt.Tags,
		&evt.Content,
		&evt.Sig,
	); err != nil {
		return nil, err
	}
	evt.CreatedAt = nostr.Timestamp(createdAt)
	return &evt, nil
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
	}{
		{tag: "e", typeName: "event"},
		{tag: "p", typeName: "pubkey"},
		{tag: "a", typeName: "address"},
		{tag: "r", typeName: "reference"},
		{tag: "t", typeName: "topic"},
	} {
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
