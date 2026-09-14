package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const upsertNIP29Group = `
INSERT INTO nip29_groups (
	relay, group_id, name, picture, about, topics, geohashes, private, closed, restricted, hidden,
	created_by, updated_at, deleted_at, min_pow, require_moderation_timeline_ref,
	min_timeline_references, timeline_recent_window, allow_late_publication,
	last_metadata_update, last_admins_update, last_members_update, last_roles_update
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
ON CONFLICT (relay, group_id) DO UPDATE SET
	name = EXCLUDED.name,
	picture = EXCLUDED.picture,
	about = EXCLUDED.about,
	topics = EXCLUDED.topics,
	geohashes = EXCLUDED.geohashes,
	private = EXCLUDED.private,
	closed = EXCLUDED.closed,
	restricted = EXCLUDED.restricted,
	hidden = EXCLUDED.hidden,
	created_by = COALESCE(nip29_groups.created_by, EXCLUDED.created_by),
	updated_at = NOW(),
	deleted_at = EXCLUDED.deleted_at,
	min_pow = EXCLUDED.min_pow,
	require_moderation_timeline_ref = EXCLUDED.require_moderation_timeline_ref,
	min_timeline_references = EXCLUDED.min_timeline_references,
	timeline_recent_window = EXCLUDED.timeline_recent_window,
	allow_late_publication = EXCLUDED.allow_late_publication,
	last_metadata_update = EXCLUDED.last_metadata_update,
	last_admins_update = EXCLUDED.last_admins_update,
	last_members_update = EXCLUDED.last_members_update,
	last_roles_update = EXCLUDED.last_roles_update
`

func (q *Queries) UpsertNIP29Group(ctx context.Context, group NIP29Group) error {
	_, err := q.db.Exec(
		ctx,
		upsertNIP29Group,
		upsertNIP29GroupArgs(group)...,
	)
	return err
}

func upsertNIP29GroupArgs(group NIP29Group) []any {
	return []any{
		group.Relay,
		group.GroupID,
		group.Name,
		group.Picture,
		group.About,
		nonNilStringSlice(group.Topics),
		nonNilStringSlice(group.Geohashes),
		group.Private,
		group.Closed,
		group.Restricted,
		group.Hidden,
		group.CreatedBy,
		group.DeletedAt,
		group.MinPoW,
		group.RequireModerationTimelineRef,
		group.MinTimelineReferences,
		group.TimelineRecentWindow,
		group.AllowLatePublication,
		group.LastMetadataUpdate,
		group.LastAdminsUpdate,
		group.LastMembersUpdate,
		group.LastRolesUpdate,
	}
}

func nonNilStringSlice(values []string) []string {
	if values != nil {
		return values
	}
	return []string{}
}

const getNIP29Group = `
SELECT relay, group_id, name, picture, about, topics, geohashes, private, closed, restricted, hidden,
	created_by, updated_at, deleted_at, min_pow, require_moderation_timeline_ref,
	min_timeline_references, timeline_recent_window, allow_late_publication,
	last_metadata_update, last_admins_update, last_members_update, last_roles_update
FROM nip29_groups
WHERE relay = $1 AND group_id = $2
`

func (q *Queries) GetNIP29Group(ctx context.Context, relay, groupID string) (*NIP29Group, bool, error) {
	row := q.db.QueryRow(ctx, getNIP29Group, relay, groupID)
	group, err := scanNIP29Group(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return group, true, nil
}

func scanNIP29Group(row interface{ Scan(...any) error }) (*NIP29Group, error) {
	var group NIP29Group
	var deletedAt *time.Time
	err := row.Scan(
		&group.Relay,
		&group.GroupID,
		&group.Name,
		&group.Picture,
		&group.About,
		&group.Topics,
		&group.Geohashes,
		&group.Private,
		&group.Closed,
		&group.Restricted,
		&group.Hidden,
		&group.CreatedBy,
		&group.UpdatedAt,
		&deletedAt,
		&group.MinPoW,
		&group.RequireModerationTimelineRef,
		&group.MinTimelineReferences,
		&group.TimelineRecentWindow,
		&group.AllowLatePublication,
		&group.LastMetadataUpdate,
		&group.LastAdminsUpdate,
		&group.LastMembersUpdate,
		&group.LastRolesUpdate,
	)
	if err != nil {
		return nil, err
	}
	group.DeletedAt = deletedAt
	return &group, nil
}

const countNIP29GroupsByCreator = `
SELECT COUNT(*)
FROM nip29_groups
WHERE relay = $1 AND created_by = $2 AND deleted_at IS NULL
`

func (q *Queries) CountNIP29GroupsByCreator(ctx context.Context, relay, createdBy string) (int, error) {
	var total int
	err := q.db.QueryRow(ctx, countNIP29GroupsByCreator, relay, createdBy).Scan(&total)
	return total, err
}

const countNIP29ActiveGroups = `
SELECT COUNT(*)
FROM nip29_groups
WHERE relay = $1 AND deleted_at IS NULL
`

func (q *Queries) CountNIP29ActiveGroups(ctx context.Context, relay string) (int, error) {
	var total int
	err := q.db.QueryRow(ctx, countNIP29ActiveGroups, relay).Scan(&total)
	return total, err
}

const listNIP29Groups = `
SELECT 
    g.relay, g.group_id, g.name, g.picture, g.about, g.topics, g.geohashes, g.private, g.closed, g.restricted, g.hidden,
    g.created_by, g.updated_at, g.deleted_at, g.min_pow, g.require_moderation_timeline_ref,
    g.min_timeline_references, g.timeline_recent_window, g.allow_late_publication,
    g.last_metadata_update, g.last_admins_update, g.last_members_update, g.last_roles_update,
    (SELECT COUNT(*) FROM nip29_group_members m WHERE m.relay = g.relay AND m.group_id = g.group_id) as member_count
FROM nip29_groups g
WHERE g.relay = $1 AND g.deleted_at IS NULL
ORDER BY g.updated_at DESC
LIMIT $2 OFFSET $3
`

const listNIP29GroupsAfter = `
SELECT g.relay, g.group_id, g.name, g.picture, g.about, g.topics, g.geohashes, g.private, g.closed, g.restricted, g.hidden,
    g.created_by, g.updated_at, g.deleted_at, g.min_pow, g.require_moderation_timeline_ref,
    g.min_timeline_references, g.timeline_recent_window, g.allow_late_publication,
    g.last_metadata_update, g.last_admins_update, g.last_members_update, g.last_roles_update,
    (SELECT COUNT(*) FROM nip29_group_members m WHERE m.relay = g.relay AND m.group_id = g.group_id) AS member_count
FROM nip29_groups g
WHERE g.relay = $1 AND g.deleted_at IS NULL
  AND ($2::timestamptz IS NULL OR g.updated_at < $2 OR (g.updated_at = $2 AND g.group_id > $3))
ORDER BY g.updated_at DESC, g.group_id ASC
LIMIT $4
`

func (q *Queries) ListNIP29Groups(ctx context.Context, relay string, limit, offset int32) ([]NIP29GroupWithMemberCount, int64, error) {
	total, err := q.CountNIP29ActiveGroups(ctx, relay)
	if err != nil {
		return nil, 0, err
	}

	rows, err := q.db.Query(ctx, listNIP29Groups, relay, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var groups []NIP29GroupWithMemberCount
	for rows.Next() {
		var g NIP29GroupWithMemberCount
		var deletedAt *time.Time
		err := rows.Scan(
			&g.Relay, &g.GroupID, &g.Name, &g.Picture, &g.About, &g.Topics, &g.Geohashes, &g.Private, &g.Closed, &g.Restricted, &g.Hidden,
			&g.CreatedBy, &g.UpdatedAt, &deletedAt, &g.MinPoW, &g.RequireModerationTimelineRef,
			&g.MinTimelineReferences, &g.TimelineRecentWindow, &g.AllowLatePublication,
			&g.LastMetadataUpdate, &g.LastAdminsUpdate, &g.LastMembersUpdate, &g.LastRolesUpdate,
			&g.MemberCount,
		)
		if err != nil {
			return nil, 0, err
		}
		g.DeletedAt = deletedAt
		groups = append(groups, g)
	}

	return groups, int64(total), nil
}

func (q *Queries) ListNIP29GroupsAfter(ctx context.Context, relay string, updatedAt *time.Time, groupID string, limit int32) ([]NIP29GroupWithMemberCount, error) {
	rows, err := q.db.Query(ctx, listNIP29GroupsAfter, relay, updatedAt, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]NIP29GroupWithMemberCount, 0, limit)
	for rows.Next() {
		var group NIP29GroupWithMemberCount
		var deletedAt *time.Time
		if err := rows.Scan(&group.Relay, &group.GroupID, &group.Name, &group.Picture, &group.About, &group.Topics, &group.Geohashes, &group.Private, &group.Closed, &group.Restricted, &group.Hidden, &group.CreatedBy, &group.UpdatedAt, &deletedAt, &group.MinPoW, &group.RequireModerationTimelineRef, &group.MinTimelineReferences, &group.TimelineRecentWindow, &group.AllowLatePublication, &group.LastMetadataUpdate, &group.LastAdminsUpdate, &group.LastMembersUpdate, &group.LastRolesUpdate, &group.MemberCount); err != nil {
			return nil, err
		}
		group.DeletedAt = deletedAt
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

type NIP29GroupWithMemberCount struct {
	NIP29Group
	MemberCount int64
}

const countNIP29GroupMessages = `
SELECT COUNT(*)
FROM event
WHERE deleted_by IS NULL
  AND kind NOT BETWEEN 9000 AND 9022
  AND kind NOT BETWEEN 39000 AND 39005
  AND tags @> $1::jsonb
`

func (q *Queries) CountNIP29GroupMessages(ctx context.Context, groupID string) (int64, error) {
	var total int64
	tag := `[["h","` + groupID + `"]]`
	err := q.db.QueryRow(ctx, countNIP29GroupMessages, tag).Scan(&total)
	return total, err
}

const getNIP29GroupCreatedAt = `
SELECT MIN(created_at)
FROM event
WHERE kind = 9007
  AND (tags @> $1::jsonb OR tags @> $2::jsonb)
`

func (q *Queries) GetNIP29GroupCreatedAt(ctx context.Context, groupID string) (int64, error) {
	var createdAt *int64
	hTag := `[["h","` + groupID + `"]]`
	dTag := `[["d","` + groupID + `"]]`
	if err := q.db.QueryRow(ctx, getNIP29GroupCreatedAt, hTag, dTag).Scan(&createdAt); err != nil {
		return 0, err
	}
	if createdAt == nil {
		return 0, nil
	}
	return *createdAt, nil
}
