package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/nbd-wtf/go-nostr"
)

type BannedUserRecord struct {
	Profile
	Reason     string    `json:"reason"`
	RelatedIDs []string  `json:"related_ids,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ReportedEventSummary struct {
	TargetEventID string   `json:"target_event_id"`
	TargetPubkey  string   `json:"target_pubkey"`
	ReportCount   int64    `json:"report_count"`
	LastReported  int64    `json:"last_reported"`
	ReportTypes   []string `json:"report_types"`
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

const getReportedEventsCountBase = `
SELECT COUNT(*)
FROM (
  SELECT target_event_id
  FROM (
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
        SELECT tag->>2
        FROM jsonb_array_elements(e.tags) tag
        WHERE tag->>0 IN ('e', 'p', 'x')
          AND COALESCE(tag->>2, '') <> ''
        LIMIT 1
      ) AS report_type
    FROM event e
    WHERE e.kind = 1984
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
) raw_reports
WHERE target_event_id IS NOT NULL
`

func appendReportedEventsFilters(base string, args []any, query string, reportType string) (string, []any) {
	query = strings.TrimSpace(query)
	if query != "" {
		args = append(args, "%"+query+"%")
		idx := len(args)
		base += fmt.Sprintf(" AND (target_event_id ILIKE $%d OR target_pubkey ILIKE $%d OR content ILIKE $%d)", idx, idx, idx)
	}

	if reportTypeValidator.MatchString(reportType) {
		args = append(args, reportType)
		base += fmt.Sprintf(" AND report_type = $%d", len(args))
	}

	return base, args
}

func (q *Queries) GetReportedEvents(ctx context.Context, query string, reportType string, limit int, offset int) ([]ReportedEventSummary, int64, error) {
	countSQL, countArgs := appendReportedEventsFilters(getReportedEventsCountBase, []any{}, query, reportType)
	countSQL += `
  GROUP BY target_event_id
) grouped_reports;
`

	var total int64
	if err := q.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL, listArgs := appendReportedEventsFilters(getReportedEventsListBase, []any{}, query, reportType)
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
