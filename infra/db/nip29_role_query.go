package db

import "context"

const ensureNIP29Role = `
INSERT INTO nip29_roles (name, description)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`

const getNIP29RoleByName = `
SELECT role_id, name, COALESCE(description, '')
FROM nip29_roles
WHERE name = $1
LIMIT 1
`

func (q *Queries) EnsureNIP29Role(ctx context.Context, name, description string) (int32, error) {
	if _, err := q.db.Exec(ctx, ensureNIP29Role, name, description); err != nil {
		return 0, err
	}

	var role NIP29Role
	err := q.db.QueryRow(ctx, getNIP29RoleByName, name).Scan(&role.RoleID, &role.Name, &role.Description)
	if err != nil {
		return 0, err
	}
	return role.RoleID, nil
}

const replaceNIP29GroupRolesDelete = `
DELETE FROM nip29_group_roles
WHERE relay = $1 AND group_id = $2
`

const insertNIP29GroupRole = `
INSERT INTO nip29_group_roles (relay, group_id, role_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`

func (q *Queries) ReplaceNIP29GroupRoles(ctx context.Context, relay, groupID string, roleIDs []int32) error {
	if _, err := q.db.Exec(ctx, replaceNIP29GroupRolesDelete, relay, groupID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if _, err := q.db.Exec(ctx, insertNIP29GroupRole, relay, groupID, roleID); err != nil {
			return err
		}
	}
	return nil
}

const listNIP29GroupRoles = `
SELECT r.role_id, r.name, COALESCE(r.description, '')
FROM nip29_group_roles gr
JOIN nip29_roles r ON r.role_id = gr.role_id
WHERE gr.relay = $1 AND gr.group_id = $2
ORDER BY r.name
`

func (q *Queries) ListNIP29GroupRoles(ctx context.Context, relay, groupID string) ([]NIP29Role, error) {
	rows, err := q.db.Query(ctx, listNIP29GroupRoles, relay, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]NIP29Role, 0, 4)
	for rows.Next() {
		var role NIP29Role
		if err := rows.Scan(&role.RoleID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
