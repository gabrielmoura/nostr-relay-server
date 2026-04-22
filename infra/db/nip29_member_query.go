package db

import (
	"context"
	"fmt"
)

const replaceNIP29MemberRolesDelete = `
DELETE FROM nip29_group_members
WHERE relay = $1 AND group_id = $2 AND user_id = $3 AND banned = FALSE
`

const insertNIP29MemberRole = `
INSERT INTO nip29_group_members (relay, group_id, user_id, role_id, banned)
VALUES ($1, $2, $3, $4, FALSE)
ON CONFLICT DO NOTHING
`

func (q *Queries) ReplaceNIP29MemberRoles(ctx context.Context, relay, groupID, userID string, roleIDs []int32) error {
	if len(roleIDs) == 0 {
		return fmt.Errorf("roleIDs cannot be empty")
	}
	if _, err := q.db.Exec(ctx, replaceNIP29MemberRolesDelete, relay, groupID, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if _, err := q.db.Exec(ctx, insertNIP29MemberRole, relay, groupID, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

const removeNIP29Member = `
DELETE FROM nip29_group_members
WHERE relay = $1 AND group_id = $2 AND user_id = $3
`

func (q *Queries) RemoveNIP29Member(ctx context.Context, relay, groupID, userID string) error {
	_, err := q.db.Exec(ctx, removeNIP29Member, relay, groupID, userID)
	return err
}

const listNIP29MemberRoles = `
SELECT gm.user_id, r.role_id, r.name, COALESCE(r.description, '')
FROM nip29_group_members gm
JOIN nip29_roles r ON r.role_id = gm.role_id
WHERE gm.relay = $1 AND gm.group_id = $2 AND gm.banned = FALSE
ORDER BY gm.user_id, r.name
`

func (q *Queries) ListNIP29MemberRoles(ctx context.Context, relay, groupID string) ([]NIP29MemberRole, error) {
	rows, err := q.db.Query(ctx, listNIP29MemberRoles, relay, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]NIP29MemberRole, 0, 8)
	for rows.Next() {
		var item NIP29MemberRole
		if err := rows.Scan(&item.UserID, &item.RoleID, &item.RoleName, &item.Description); err != nil {
			return nil, err
		}
		members = append(members, item)
	}
	return members, rows.Err()
}

const isNIP29Member = `
SELECT EXISTS (
	SELECT 1
	FROM nip29_group_members
	WHERE relay = $1 AND group_id = $2 AND user_id = $3 AND banned = FALSE
)
`

func (q *Queries) IsNIP29Member(ctx context.Context, relay, groupID, userID string) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, isNIP29Member, relay, groupID, userID).Scan(&exists)
	return exists, err
}

const getNIP29MemberRoleNames = `
SELECT r.name
FROM nip29_group_members gm
JOIN nip29_roles r ON r.role_id = gm.role_id
WHERE gm.relay = $1 AND gm.group_id = $2 AND gm.user_id = $3 AND gm.banned = FALSE
ORDER BY r.name
`

func (q *Queries) GetNIP29MemberRoleNames(ctx context.Context, relay, groupID, userID string) ([]string, error) {
	rows, err := q.db.Query(ctx, getNIP29MemberRoleNames, relay, groupID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]string, 0, 4)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
}
