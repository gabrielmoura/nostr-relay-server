package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const upsertNIP29Group = `
INSERT INTO nip29_groups (
	relay, group_id, name, picture, about, private, closed, restricted, hidden,
	created_by, updated_at, deleted_at, min_pow, require_moderation_timeline_ref,
	min_timeline_references, timeline_recent_window, allow_late_publication,
	last_metadata_update, last_admins_update, last_members_update, last_roles_update
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
ON CONFLICT (relay, group_id) DO UPDATE SET
	name = EXCLUDED.name,
	picture = EXCLUDED.picture,
	about = EXCLUDED.about,
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
		group.Relay,
		group.GroupID,
		group.Name,
		group.Picture,
		group.About,
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
	)
	return err
}

const getNIP29Group = `
SELECT relay, group_id, name, picture, about, private, closed, restricted, hidden,
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
    g.relay, g.group_id, g.name, g.picture, g.about, g.private, g.closed, g.restricted, g.hidden,
    g.created_by, g.updated_at, g.deleted_at, g.min_pow, g.require_moderation_timeline_ref,
    g.min_timeline_references, g.timeline_recent_window, g.allow_late_publication,
    g.last_metadata_update, g.last_admins_update, g.last_members_update, g.last_roles_update,
    (SELECT COUNT(*) FROM nip29_group_members m WHERE m.relay = g.relay AND m.group_id = g.group_id) as member_count
FROM nip29_groups g
WHERE g.relay = $1 AND g.deleted_at IS NULL
ORDER BY g.updated_at DESC
LIMIT $2 OFFSET $3
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
			&g.Relay, &g.GroupID, &g.Name, &g.Picture, &g.About, &g.Private, &g.Closed, &g.Restricted, &g.Hidden,
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

type NIP29GroupWithMemberCount struct {
	NIP29Group
	MemberCount int64
}
