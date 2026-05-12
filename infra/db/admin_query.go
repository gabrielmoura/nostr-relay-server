package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/nbd-wtf/go-nostr"
)

type BannedUserRecord struct {
	Profile
	Reason     string       `json:"reason"`
	RelatedIDs []string     `json:"related_ids,omitempty"`
	CreatedAt  sql.NullTime `json:"created_at"`
}

type ReportedEventSummary struct {
	TargetEventID string   `json:"target_event_id"`
	TargetPubkey  string   `json:"target_pubkey"`
	ReportCount   int64    `json:"report_count"`
	LastReported  int64    `json:"last_reported"`
	ReportTypes   []string `json:"report_types"`
}

type ReportedTimelinePoint struct {
	Bucket string `json:"bucket"`
	Count  int64  `json:"count"`
}

type ReportedTypeCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type ReportedAuthorCount struct {
	Pubkey string `json:"pubkey"`
	Count  int64  `json:"count"`
}

type ReportedTargetCount struct {
	TargetEventID string `json:"target_event_id"`
	Count         int64  `json:"count"`
}

type ReportedEventsFilters struct {
	Query         string
	ReportType    string
	TargetPubkey  string
	TargetEventID string
	Since         int64
	Until         int64
}

type ReportedEventsSummary struct {
	TotalEvents         int64                   `json:"total_events"`
	TotalReports        int64                   `json:"total_reports"`
	UniqueTargetAuthors int64                   `json:"unique_target_authors"`
	Timeline            []ReportedTimelinePoint `json:"timeline"`
	ReportTypes         []ReportedTypeCount     `json:"report_types"`
	TopAuthors          []ReportedAuthorCount   `json:"top_authors"`
	TopTargets          []ReportedTargetCount   `json:"top_targets"`
}

type EventKindAggregate struct {
	Kind  int   `json:"kind"`
	Count int64 `json:"count"`
}

type EventAuthorAggregate struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"display_name,omitempty"`
	Count       int64  `json:"count"`
}

type EventTagAggregate struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type EventTrendAggregate struct {
	TopTagMonth      string `json:"top_tag_month,omitempty"`
	TopTagMonthCount int64  `json:"top_tag_month_count,omitempty"`
	TopTagYear       string `json:"top_tag_year,omitempty"`
	TopTagYearCount  int64  `json:"top_tag_year_count,omitempty"`
	PeakMonth        string `json:"peak_month,omitempty"`
	PeakMonthCount   int64  `json:"peak_month_count,omitempty"`
	PeakYear         string `json:"peak_year,omitempty"`
	PeakYearCount    int64  `json:"peak_year_count,omitempty"`
}

type EventTimelinePoint struct {
	TS    int64 `json:"ts"`
	Count int64 `json:"count"`
}

type EventAggregates struct {
	Total      int64                  `json:"total"`
	Kinds      []EventKindAggregate   `json:"kinds"`
	TopAuthors []EventAuthorAggregate `json:"top_authors"`
	TopTags    []EventTagAggregate    `json:"top_tags"`
	Trends     EventTrendAggregate    `json:"trends"`
}

const countAllEvents = `-- name: CountAllEvents :one
SELECT COUNT(*) FROM event;
`

func (q *Queries) CountAllEvents(ctx context.Context) (int64, error) {
	var count int64
	if err := q.db.QueryRow(ctx, countAllEvents).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

const countEventsSince = `-- name: CountEventsSince :one
SELECT COUNT(*)
FROM event
WHERE created_at >= $1::bigint;
`

func (q *Queries) CountEventsSince(ctx context.Context, since int64) (int64, error) {
	var count int64
	if err := q.db.QueryRow(ctx, countEventsSince, since).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) QueryEventsWindow(ctx context.Context, filter nostr.Filter, offset int) ([]*nostr.Event, int64, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	query, params, err := helper.QueryEventsSql(&config.Cfg.Relay, filter, false)
	if err != nil {
		return nil, 0, err
	}
	query = query + fmt.Sprintf(" OFFSET $%d", len(params)+1)
	params = append(params, offset)

	total, err := q.CountEvents(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	rows, err := q.db.Query(ctx, query, params...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, 0, fmt.Errorf("failed to fetch admin events using query %q: %w", query, err)
	}
	if err != nil {
		return []*nostr.Event{}, total, nil
	}
	defer rows.Close()

	events := make([]*nostr.Event, 0, filter.Limit)
	for rows.Next() {
		var evt nostr.Event
		var timestamp int64
		if err := rows.Scan(&evt.ID, &evt.PubKey, &timestamp, &evt.Kind, &evt.Tags, &evt.Content, &evt.Sig); err != nil {
			return nil, 0, err
		}
		evt.CreatedAt = nostr.Timestamp(timestamp)
		events = append(events, &evt)
	}

	return events, total, nil
}

func (q *Queries) GetEventAggregates(ctx context.Context, filter nostr.Filter) (EventAggregates, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	whereClause, params := helper.BuildWhereClause(filter, &config.Cfg.Relay)

	total, err := q.CountEvents(ctx, filter)
	if err != nil {
		return EventAggregates{}, err
	}

	aggregates := EventAggregates{Total: total}

	kindsSQL := fmt.Sprintf(`
SELECT kind, COUNT(*) AS count
FROM event
WHERE %s
GROUP BY kind
ORDER BY count DESC, kind ASC
LIMIT 8;`, whereClause)

	kindRows, err := q.db.Query(ctx, kindsSQL, params...)
	if err != nil {
		return EventAggregates{}, err
	}
	defer kindRows.Close()
	for kindRows.Next() {
		var item EventKindAggregate
		if err := kindRows.Scan(&item.Kind, &item.Count); err != nil {
			return EventAggregates{}, err
		}
		aggregates.Kinds = append(aggregates.Kinds, item)
	}

	authorsSQL := fmt.Sprintf(`
SELECT event.pubkey,
       COALESCE(NULLIF(profiles.display_name, ''), NULLIF(profiles.name, ''), event.pubkey) AS display_name,
       COUNT(*) AS count
FROM event
LEFT JOIN profiles ON profiles.public_key = event.pubkey
WHERE %s
GROUP BY event.pubkey, profiles.display_name, profiles.name
ORDER BY count DESC, event.pubkey ASC
LIMIT 8;`, whereClause)

	authorRows, err := q.db.Query(ctx, authorsSQL, params...)
	if err != nil {
		return EventAggregates{}, err
	}
	defer authorRows.Close()
	for authorRows.Next() {
		var item EventAuthorAggregate
		if err := authorRows.Scan(&item.Pubkey, &item.DisplayName, &item.Count); err != nil {
			return EventAggregates{}, err
		}
		aggregates.TopAuthors = append(aggregates.TopAuthors, item)
	}

	tagsSQL := fmt.Sprintf(`
SELECT LOWER(BTRIM(tag->>1)) AS tag, COUNT(*) AS count
FROM event
CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
WHERE %s
  AND jsonb_typeof(tag) = 'array'
  AND jsonb_array_length(tag) >= 2
  AND tag->>0 = 't'
  AND BTRIM(tag->>1) <> ''
GROUP BY LOWER(BTRIM(tag->>1))
ORDER BY count DESC, tag ASC
LIMIT 12;`, whereClause)

	tagRows, err := q.db.Query(ctx, tagsSQL, params...)
	if err != nil {
		return EventAggregates{}, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var item EventTagAggregate
		if err := tagRows.Scan(&item.Tag, &item.Count); err != nil {
			return EventAggregates{}, err
		}
		aggregates.TopTags = append(aggregates.TopTags, item)
	}

	trendsSQL := fmt.Sprintf(`
WITH filtered AS (
    SELECT created_at, tags
    FROM event
    WHERE %s
),
month_tags AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY-MM') AS period,
           LOWER(BTRIM(tag->>1)) AS tag,
           COUNT(*) AS count
    FROM filtered
    CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
    WHERE jsonb_typeof(tag) = 'array'
      AND jsonb_array_length(tag) >= 2
      AND tag->>0 = 't'
      AND BTRIM(tag->>1) <> ''
    GROUP BY period, LOWER(BTRIM(tag->>1))
    ORDER BY count DESC, period ASC, tag ASC
    LIMIT 1
),
year_tags AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY') AS period,
           LOWER(BTRIM(tag->>1)) AS tag,
           COUNT(*) AS count
    FROM filtered
    CROSS JOIN LATERAL jsonb_array_elements(tags) AS tag
    WHERE jsonb_typeof(tag) = 'array'
      AND jsonb_array_length(tag) >= 2
      AND tag->>0 = 't'
      AND BTRIM(tag->>1) <> ''
    GROUP BY period, LOWER(BTRIM(tag->>1))
    ORDER BY count DESC, period ASC, tag ASC
    LIMIT 1
),
month_counts AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY-MM') AS period,
           COUNT(*) AS count
    FROM filtered
    GROUP BY period
    ORDER BY count DESC, period ASC
    LIMIT 1
),
year_counts AS (
    SELECT to_char(to_timestamp(created_at), 'YYYY') AS period,
           COUNT(*) AS count
    FROM filtered
    GROUP BY period
    ORDER BY count DESC, period ASC
    LIMIT 1
)
SELECT COALESCE((SELECT period || ' · ' || tag FROM month_tags), '') AS top_tag_month,
       COALESCE((SELECT count FROM month_tags), 0) AS top_tag_month_count,
       COALESCE((SELECT period || ' · ' || tag FROM year_tags), '') AS top_tag_year,
       COALESCE((SELECT count FROM year_tags), 0) AS top_tag_year_count,
       COALESCE((SELECT period FROM month_counts), '') AS peak_month,
       COALESCE((SELECT count FROM month_counts), 0) AS peak_month_count,
       COALESCE((SELECT period FROM year_counts), '') AS peak_year,
       COALESCE((SELECT count FROM year_counts), 0) AS peak_year_count;`, whereClause)

	if err := q.db.QueryRow(ctx, trendsSQL, params...).Scan(
		&aggregates.Trends.TopTagMonth,
		&aggregates.Trends.TopTagMonthCount,
		&aggregates.Trends.TopTagYear,
		&aggregates.Trends.TopTagYearCount,
		&aggregates.Trends.PeakMonth,
		&aggregates.Trends.PeakMonthCount,
		&aggregates.Trends.PeakYear,
		&aggregates.Trends.PeakYearCount,
	); err != nil {
		return EventAggregates{}, err
	}

	return aggregates, nil
}

func (q *Queries) GetEventTimeline(ctx context.Context, filter nostr.Filter, bucket string) ([]EventTimelinePoint, error) {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	whereClause, params := helper.BuildWhereClause(filter, &config.Cfg.Relay)

	step := int64(3600)
	if strings.EqualFold(strings.TrimSpace(bucket), "day") {
		step = 86400
	}

	timelineSQL := fmt.Sprintf(`
SELECT (created_at / %s) * %s AS ts, COUNT(*) AS count
FROM event
WHERE %s
GROUP BY ts
ORDER BY ts ASC;`, strconv.FormatInt(step, 10), strconv.FormatInt(step, 10), whereClause)

	rows, err := q.db.Query(ctx, timelineSQL, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]EventTimelinePoint, 0, 32)
	for rows.Next() {
		var item EventTimelinePoint
		if err := rows.Scan(&item.TS, &item.Count); err != nil {
			return nil, err
		}
		points = append(points, item)
	}

	return points, nil
}

const getProfileByPublicKey = `-- name: GetProfileByPublicKey :one
SELECT id, public_key, name, about, picture, banner, website, display_name, lud16, pronouns, nip05, bot
FROM profiles
WHERE public_key = $1::text
LIMIT 1;
`

func (q *Queries) GetProfileByPublicKey(ctx context.Context, key string) (Profile, error) {
	if cached, ok := cache.GetProfile(key); ok {
		return Profile{
			PublicKey:   key,
			Name:        cached.Name,
			DisplayName: cached.DisplayName,
			About:       cached.About,
			Picture:     cached.Picture,
			Website:     cached.Website,
			Nip05:       cached.NIP05,
			Lud16:       cached.LUD16,
			Bot:         cached.Bot,
		}, nil
	}

	var profile Profile
	err := q.db.QueryRow(ctx, getProfileByPublicKey, key).Scan(
		&profile.ID,
		&profile.PublicKey,
		&profile.Name,
		&profile.About,
		&profile.Picture,
		&profile.Banner,
		&profile.Website,
		&profile.DisplayName,
		&profile.Lud16,
		&profile.Pronouns,
		&profile.Nip05,
		&profile.Bot,
	)
	if err != nil {
		return Profile{}, err
	}
	_ = cache.SetProfile(key, &cache.ProfileCache{
		Name:        profile.Name,
		DisplayName: profile.DisplayName,
		About:       profile.About,
		Picture:     profile.Picture,
		Website:     profile.Website,
		NIP05:       profile.Nip05,
		LUD16:       profile.Lud16,
		Bot:         profile.Bot,
	})
	return profile, nil
}

const getProfilesByPublicKeys = `-- name: GetProfilesByPublicKeys :many
SELECT id, public_key, name, about, picture, banner, website, display_name, lud16, pronouns, nip05, bot
FROM profiles
WHERE public_key = ANY($1::text[]);
`

func (q *Queries) GetProfilesByPublicKeys(ctx context.Context, keys []string) (map[string]Profile, error) {
	profiles := make(map[string]Profile, len(keys))
	if len(keys) == 0 {
		return profiles, nil
	}

	rows, err := q.db.Query(ctx, getProfilesByPublicKeys, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var profile Profile
		if err := rows.Scan(
			&profile.ID,
			&profile.PublicKey,
			&profile.Name,
			&profile.About,
			&profile.Picture,
			&profile.Banner,
			&profile.Website,
			&profile.DisplayName,
			&profile.Lud16,
			&profile.Pronouns,
			&profile.Nip05,
			&profile.Bot,
		); err != nil {
			return nil, err
		}
		profiles[profile.PublicKey] = profile
	}

	return profiles, nil
}

func buildAdminProfileSearchQuery(base string, query string) (string, []any) {
	query = strings.TrimSpace(query)
	if query == "" {
		return base, []any{}
	}

	if len(query) == 64 {
		return base + `
WHERE public_key = $1
   OR public_key ILIKE $2
   OR name ILIKE $2
   OR display_name ILIKE $2
   OR nip05 ILIKE $2`, []any{query, "%" + query + "%"}
	}

	needle := "%" + query + "%"
	return base + `
WHERE public_key ILIKE $1
   OR name ILIKE $1
   OR display_name ILIKE $1
   OR nip05 ILIKE $1`, []any{needle}
}

func scanProfile(row scanner) (Profile, error) {
	var profile Profile
	err := row.Scan(
		&profile.ID,
		&profile.PublicKey,
		&profile.Name,
		&profile.About,
		&profile.Picture,
		&profile.Banner,
		&profile.Website,
		&profile.DisplayName,
		&profile.Lud16,
		&profile.Pronouns,
		&profile.Nip05,
		&profile.Bot,
	)
	return profile, err
}

type scanner interface {
	Scan(dest ...any) error
}

const searchProfilesBase = `
SELECT id, public_key, name, about, picture, banner, website, display_name, lud16, pronouns, nip05, bot
FROM profiles
`

func (q *Queries) SearchProfiles(ctx context.Context, query string, limit int, offset int) ([]Profile, int64, error) {
	countQuery, countArgs := buildAdminProfileSearchQuery(`SELECT COUNT(*) FROM profiles`, query)
	var total int64
	if err := q.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery, listArgs := buildAdminProfileSearchQuery(searchProfilesBase, query)
	listArgs = append(listArgs, limit, offset)
	listQuery += fmt.Sprintf(" ORDER BY COALESCE(display_name, name, public_key), public_key LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	profiles := make([]Profile, 0, limit)
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, 0, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, total, nil
}

const listBannedUsersBase = `
SELECT p.id, p.public_key, p.name, p.about, p.picture, p.banner, p.website, p.display_name, p.lud16, p.pronouns, p.nip05, p.bot,
       b.reason, COALESCE(b.related_ids, '{}'), NULL::timestamptz AS created_at
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, related_ids, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
`

const countBannedUsersBase = `
SELECT COUNT(*)
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, related_ids, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
`

func buildBannedUsersQuery(base string, query string) (string, []any) {
	query = strings.TrimSpace(query)
	if query == "" {
		return base, []any{}
	}

	needle := "%" + query + "%"
	return base + `
WHERE p.public_key ILIKE $1
   OR p.name ILIKE $1
   OR p.display_name ILIKE $1
   OR p.nip05 ILIKE $1
   OR b.reason ILIKE $1`, []any{needle}
}

func (q *Queries) ListBannedUsers(ctx context.Context, query string, limit int, offset int) ([]BannedUserRecord, int64, error) {
	countQuery, countArgs := buildBannedUsersQuery(countBannedUsersBase, query)
	var total int64
	if err := q.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery, listArgs := buildBannedUsersQuery(listBannedUsersBase, query)
	listArgs = append(listArgs, limit, offset)
	listQuery += fmt.Sprintf(" ORDER BY p.public_key LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]BannedUserRecord, 0, limit)
	for rows.Next() {
		var item BannedUserRecord
		if err := rows.Scan(
			&item.ID,
			&item.PublicKey,
			&item.Name,
			&item.About,
			&item.Picture,
			&item.Banner,
			&item.Website,
			&item.DisplayName,
			&item.Lud16,
			&item.Pronouns,
			&item.Nip05,
			&item.Bot,
			&item.Reason,
			&item.RelatedIDs,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

const getLatestBanRecordByKey = `-- name: GetLatestBanRecordByKey :one
SELECT b.reason, COALESCE(b.related_ids, '{}'), NULL::timestamptz AS created_at
FROM banned_users b
JOIN profiles p ON b.user_id = p.id
WHERE p.public_key = $1::text
ORDER BY b.id DESC
LIMIT 1;
`

func (q *Queries) GetLatestBanRecordByKey(ctx context.Context, key string) (BannedUserRecord, bool, error) {
	profile, err := q.GetProfileByPublicKey(ctx, key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return BannedUserRecord{}, false, err
	}

	var record BannedUserRecord
	var relatedIDs []string
	err = q.db.QueryRow(ctx, getLatestBanRecordByKey, key).Scan(&record.Reason, &relatedIDs, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BannedUserRecord{}, false, nil
	}
	if err != nil {
		return BannedUserRecord{}, false, err
	}

	record.Profile = profile
	record.RelatedIDs = relatedIDs
	return record, true, nil
}

const getLatestBanRecordsByKeys = `-- name: GetLatestBanRecordsByKeys :many
SELECT p.public_key, b.reason, COALESCE(b.related_ids, '{}'), NULL::timestamptz AS created_at
FROM (
    SELECT DISTINCT ON (user_id) user_id, reason, related_ids, id
    FROM banned_users
    ORDER BY user_id, id DESC
) b
JOIN profiles p ON p.id = b.user_id
WHERE p.public_key = ANY($1::text[]);
`

func (q *Queries) GetLatestBanRecordsByKeys(ctx context.Context, keys []string) (map[string]BannedUserRecord, error) {
	records := make(map[string]BannedUserRecord, len(keys))
	if len(keys) == 0 {
		return records, nil
	}

	rows, err := q.db.Query(ctx, getLatestBanRecordsByKeys, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pubkey string
		var record BannedUserRecord
		if err := rows.Scan(&pubkey, &record.Reason, &record.RelatedIDs, &record.CreatedAt); err != nil {
			return nil, err
		}
		records[pubkey] = record
	}

	return records, nil
}

const getEventByID = `-- name: GetEventByID :one
SELECT id, pubkey, created_at, kind, tags, content, sig
FROM event
WHERE id = $1::text
LIMIT 1;
`

func (q *Queries) GetEventByID(ctx context.Context, id string) (*nostr.Event, error) {
	var evt nostr.Event
	var createdAt int64
	err := q.db.QueryRow(ctx, getEventByID, id).Scan(
		&evt.ID,
		&evt.PubKey,
		&createdAt,
		&evt.Kind,
		&evt.Tags,
		&evt.Content,
		&evt.Sig,
	)
	if err != nil {
		return nil, err
	}
	evt.CreatedAt = nostr.Timestamp(createdAt)
	return &evt, nil
}

const getReportsForEventCount = `-- name: GetReportsForEventCount :one
SELECT COUNT(*)
FROM event e
WHERE e.kind = 1984
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(e.tags) tag
    WHERE tag->>0 = 'e'
      AND tag->>1 = $1::text
  );
`

const getReportsForEvent = `-- name: GetReportsForEvent :many
SELECT e.id, e.pubkey, e.created_at, e.kind, e.tags, e.content, e.sig
FROM event e
WHERE e.kind = 1984
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(e.tags) tag
    WHERE tag->>0 = 'e'
      AND tag->>1 = $1::text
  )
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3;
`

func (q *Queries) GetReportsForEvent(ctx context.Context, eventID string, limit int, offset int) ([]*nostr.Event, int64, error) {
	var total int64
	if err := q.db.QueryRow(ctx, getReportsForEventCount, eventID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := q.db.Query(ctx, getReportsForEvent, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*nostr.Event, 0, limit)
	for rows.Next() {
		var evt nostr.Event
		var createdAt int64
		if err := rows.Scan(
			&evt.ID,
			&evt.PubKey,
			&createdAt,
			&evt.Kind,
			&evt.Tags,
			&evt.Content,
			&evt.Sig,
		); err != nil {
			return nil, 0, err
		}
		evt.CreatedAt = nostr.Timestamp(createdAt)
		items = append(items, &evt)
	}

	return items, total, nil
}

var reportTypeValidator = regexp.MustCompile(`^(nudity|malware|profanity|illegal|spam|impersonation|other)$`)

const getReportedEventsRawBase = `
SELECT
  e.id,
  e.content,
  e.created_at,
  (
    SELECT tag->>1
    FROM jsonb_array_elements(e.tags) tag
    WHERE tag->>0 = 'e'
    LIMIT 1
  ) AS target_event_id,
  (
    SELECT tag->>1
    FROM jsonb_array_elements(e.tags) tag
    WHERE tag->>0 = 'p'
    LIMIT 1
  ) AS target_pubkey,
  (
    SELECT tag->>2
    FROM jsonb_array_elements(e.tags) tag
    WHERE tag->>0 IN ('e', 'p', 'x')
      AND COALESCE(tag->>2, '') <> ''
    LIMIT 1
  ) AS report_type
FROM event e
WHERE e.kind = 1984
`

const getReportedEventsFilteredBase = `
SELECT
  id,
  content,
  created_at,
  target_event_id,
  target_pubkey,
  report_type
FROM (
` + getReportedEventsRawBase + `
) raw_reports
WHERE target_event_id IS NOT NULL
`

const getReportedEventsListBase = `
SELECT
  target_event_id,
  COALESCE(
    MAX(target_pubkey) FILTER (WHERE target_pubkey IS NOT NULL),
    ''
  ) AS target_pubkey,
  COUNT(*) AS report_count,
  MAX(created_at) AS last_reported,
  ARRAY_REMOVE(ARRAY_AGG(DISTINCT report_type), NULL) AS report_types
FROM (
` + getReportedEventsFilteredBase + `
) filtered_reports
WHERE target_event_id IS NOT NULL
`

func appendReportedEventsFilters(base string, args []any, filters ReportedEventsFilters) (string, []any) {
	query := strings.TrimSpace(filters.Query)
	if query != "" {
		args = append(args, "%"+query+"%")
		idx := len(args)
		base += fmt.Sprintf(" AND (target_event_id ILIKE $%d OR target_pubkey ILIKE $%d OR content ILIKE $%d)", idx, idx, idx)
	}

	if reportTypeValidator.MatchString(filters.ReportType) {
		args = append(args, filters.ReportType)
		base += fmt.Sprintf(" AND report_type = $%d", len(args))
	}

	if value := strings.TrimSpace(filters.TargetPubkey); value != "" {
		args = append(args, value)
		base += fmt.Sprintf(" AND target_pubkey = $%d", len(args))
	}

	if value := strings.TrimSpace(filters.TargetEventID); value != "" {
		args = append(args, value)
		base += fmt.Sprintf(" AND target_event_id = $%d", len(args))
	}

	if filters.Since > 0 {
		args = append(args, filters.Since)
		base += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}

	if filters.Until > 0 {
		args = append(args, filters.Until)
		base += fmt.Sprintf(" AND created_at <= $%d", len(args))
	}

	return base, args
}

func (q *Queries) GetReportedEvents(ctx context.Context, filters ReportedEventsFilters, limit int, offset int) ([]ReportedEventSummary, int64, error) {
	countSQL, countArgs := appendReportedEventsFilters(getReportedEventsFilteredBase, []any{}, filters)
	countSQL = `SELECT COUNT(DISTINCT target_event_id) FROM (` + countSQL + `
) grouped_reports;`

	var total int64
	if err := q.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL, listArgs := appendReportedEventsFilters(getReportedEventsListBase, []any{}, filters)
	listArgs = append(listArgs, limit, offset)
	listSQL += fmt.Sprintf(`
GROUP BY target_event_id
ORDER BY last_reported DESC
LIMIT $%d OFFSET $%d;`, len(listArgs)-1, len(listArgs))

	rows, err := q.db.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ReportedEventSummary, 0, limit)
	for rows.Next() {
		var item ReportedEventSummary
		if err := rows.Scan(&item.TargetEventID, &item.TargetPubkey, &item.ReportCount, &item.LastReported, &item.ReportTypes); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (q *Queries) GetReportedEventsSummary(ctx context.Context, filters ReportedEventsFilters) (ReportedEventsSummary, error) {
	groupedSQL, groupedArgs := appendReportedEventsFilters(getReportedEventsListBase, []any{}, filters)
	summarySQL := `WITH grouped_reports AS (` + groupedSQL + `
GROUP BY target_event_id
)
SELECT
  COUNT(*) AS total_events,
  COALESCE(SUM(report_count), 0) AS total_reports,
  COUNT(DISTINCT NULLIF(target_pubkey, '')) AS unique_target_authors
FROM grouped_reports;
`

	var summary ReportedEventsSummary
	if err := q.db.QueryRow(ctx, summarySQL, groupedArgs...).Scan(&summary.TotalEvents, &summary.TotalReports, &summary.UniqueTargetAuthors); err != nil {
		return ReportedEventsSummary{}, err
	}

	timelineSQL := `WITH grouped_reports AS (` + groupedSQL + `
GROUP BY target_event_id
)
SELECT to_char(to_timestamp(last_reported), 'YYYY-MM-DD') AS bucket, COALESCE(SUM(report_count), 0) AS count
FROM grouped_reports
GROUP BY bucket
ORDER BY bucket ASC;
`

	timelineRows, err := q.db.Query(ctx, timelineSQL, groupedArgs...)
	if err != nil {
		return ReportedEventsSummary{}, err
	}
	defer timelineRows.Close()
	for timelineRows.Next() {
		var item ReportedTimelinePoint
		if err := timelineRows.Scan(&item.Bucket, &item.Count); err != nil {
			return ReportedEventsSummary{}, err
		}
		summary.Timeline = append(summary.Timeline, item)
	}

	rawSQL, rawArgs := appendReportedEventsFilters(getReportedEventsFilteredBase, []any{}, filters)
	typesSQL := `WITH filtered_reports AS (` + rawSQL + `
)
SELECT report_type AS name, COUNT(*) AS count
FROM filtered_reports
WHERE report_type IS NOT NULL AND report_type <> ''
GROUP BY report_type
ORDER BY count DESC, name ASC;
`

	typeRows, err := q.db.Query(ctx, typesSQL, rawArgs...)
	if err != nil {
		return ReportedEventsSummary{}, err
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var item ReportedTypeCount
		if err := typeRows.Scan(&item.Name, &item.Count); err != nil {
			return ReportedEventsSummary{}, err
		}
		summary.ReportTypes = append(summary.ReportTypes, item)
	}

	authorsSQL := `WITH filtered_reports AS (` + rawSQL + `
)
SELECT target_pubkey AS pubkey, COUNT(*) AS count
FROM filtered_reports
WHERE target_pubkey IS NOT NULL AND target_pubkey <> ''
GROUP BY target_pubkey
ORDER BY count DESC, pubkey ASC
LIMIT 8;
`

	authorRows, err := q.db.Query(ctx, authorsSQL, rawArgs...)
	if err != nil {
		return ReportedEventsSummary{}, err
	}
	defer authorRows.Close()
	for authorRows.Next() {
		var item ReportedAuthorCount
		if err := authorRows.Scan(&item.Pubkey, &item.Count); err != nil {
			return ReportedEventsSummary{}, err
		}
		summary.TopAuthors = append(summary.TopAuthors, item)
	}

	targetsSQL := `WITH grouped_reports AS (` + groupedSQL + `
GROUP BY target_event_id
)
SELECT target_event_id, report_count AS count
FROM grouped_reports
ORDER BY report_count DESC, target_event_id ASC
LIMIT 8;
`

	targetRows, err := q.db.Query(ctx, targetsSQL, groupedArgs...)
	if err != nil {
		return ReportedEventsSummary{}, err
	}
	defer targetRows.Close()
	for targetRows.Next() {
		var item ReportedTargetCount
		if err := targetRows.Scan(&item.TargetEventID, &item.Count); err != nil {
			return ReportedEventsSummary{}, err
		}
		summary.TopTargets = append(summary.TopTargets, item)
	}

	return summary, nil
}
