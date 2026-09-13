package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const getNIP29Parent = `
SELECT parent_group_id
FROM nip29_group_hierarchy
WHERE relay = $1 AND child_group_id = $2
`

func (q *Queries) GetNIP29Parent(ctx context.Context, relay, childGroupID string) (string, bool, error) {
	var parentGroupID string
	err := q.db.QueryRow(ctx, getNIP29Parent, relay, childGroupID).Scan(&parentGroupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return parentGroupID, true, nil
}

const listNIP29Children = `
SELECT child_group_id
FROM nip29_group_hierarchy
WHERE relay = $1 AND parent_group_id = $2
ORDER BY position
`

func (q *Queries) ListNIP29Children(ctx context.Context, relay, parentGroupID string) ([]string, error) {
	rows, err := q.db.Query(ctx, listNIP29Children, relay, parentGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := make([]string, 0, 4)
	for rows.Next() {
		var childGroupID string
		if err := rows.Scan(&childGroupID); err != nil {
			return nil, err
		}
		children = append(children, childGroupID)
	}
	return children, rows.Err()
}

const setNIP29Parent = `
INSERT INTO nip29_group_hierarchy (relay, child_group_id, parent_group_id, position)
VALUES ($1, $2, $3, COALESCE((SELECT MAX(position) + 1 FROM nip29_group_hierarchy WHERE relay = $1 AND parent_group_id = $3), 0))
ON CONFLICT (relay, child_group_id) DO UPDATE SET parent_group_id = EXCLUDED.parent_group_id, position = EXCLUDED.position
`

const deleteNIP29Parent = `
DELETE FROM nip29_group_hierarchy WHERE relay = $1 AND child_group_id = $2
`

func (q *Queries) SetNIP29Parent(ctx context.Context, relay, childGroupID, parentGroupID string) error {
	if parentGroupID == "" {
		_, err := q.db.Exec(ctx, deleteNIP29Parent, relay, childGroupID)
		return err
	}
	_, err := q.db.Exec(ctx, setNIP29Parent, relay, childGroupID, parentGroupID)
	return err
}

const replaceNIP29ChildOrderDelete = `
DELETE FROM nip29_group_hierarchy WHERE relay = $1 AND parent_group_id = $2
`

const insertNIP29ChildOrder = `
INSERT INTO nip29_group_hierarchy (relay, child_group_id, parent_group_id, position)
VALUES ($1, $2, $3, $4)
`

func (q *Queries) ReplaceNIP29ChildOrder(ctx context.Context, relay, parentGroupID string, children []string) error {
	if _, err := q.db.Exec(ctx, replaceNIP29ChildOrderDelete, relay, parentGroupID); err != nil {
		return err
	}
	for position, childGroupID := range children {
		if _, err := q.db.Exec(ctx, insertNIP29ChildOrder, relay, childGroupID, parentGroupID, position); err != nil {
			return err
		}
	}
	return nil
}
