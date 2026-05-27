package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type BlossomPlanAssignment struct {
	PlanID      string
	Pubkey      string
	Name        sql.NullString
	DisplayName sql.NullString
	Picture     sql.NullString
	AssignedBy  string
	AssignedAt  time.Time
}

func (q *Queries) ListPlanAssignments(ctx context.Context, planID string) ([]BlossomPlanAssignment, error) {
	const statement = `
SELECT
	pa.plan_id,
	pa.pubkey,
	p.name,
	p.display_name,
	p.picture,
	pa.assigned_by,
	pa.assigned_at
FROM blossom_plan_assignments pa
LEFT JOIN profiles p ON p.public_key = pa.pubkey
WHERE pa.plan_id = $1::text
ORDER BY pa.assigned_at DESC, pa.pubkey ASC`

	rows, err := q.db.Query(ctx, statement, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BlossomPlanAssignment, 0, 16)
	for rows.Next() {
		var item BlossomPlanAssignment
		if err := rows.Scan(
			&item.PlanID,
			&item.Pubkey,
			&item.Name,
			&item.DisplayName,
			&item.Picture,
			&item.AssignedBy,
			&item.AssignedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (q *Queries) AssignPlanToUser(ctx context.Context, planID string, pubkey string, assignedBy string) error {
	const deleteStatement = `DELETE FROM blossom_plan_assignments WHERE pubkey = $1::text`
	if _, err := q.db.Exec(ctx, deleteStatement, pubkey); err != nil {
		return err
	}

	const insertStatement = `
INSERT INTO blossom_plan_assignments (plan_id, pubkey, assigned_by, assigned_at)
VALUES ($1::text, $2::text, $3::text, NOW())`
	_, err := q.db.Exec(ctx, insertStatement, planID, pubkey, assignedBy)
	return err
}

func (q *Queries) UnassignPlanFromUser(ctx context.Context, planID string, pubkey string) error {
	const statement = `DELETE FROM blossom_plan_assignments WHERE plan_id = $1::text AND pubkey = $2::text`
	_, err := q.db.Exec(ctx, statement, planID, pubkey)
	return err
}

func (q *Queries) GetUserPlanAssignment(ctx context.Context, pubkey string) (BlossomPlanAssignment, bool, error) {
	const statement = `SELECT plan_id, pubkey, assigned_by, assigned_at FROM blossom_plan_assignments WHERE pubkey = $1::text LIMIT 1`
	var item BlossomPlanAssignment
	err := q.db.QueryRow(ctx, statement, pubkey).Scan(
		&item.PlanID,
		&item.Pubkey,
		&item.AssignedBy,
		&item.AssignedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BlossomPlanAssignment{}, false, nil
		}
		return BlossomPlanAssignment{}, false, err
	}
	return item, true, nil
}
