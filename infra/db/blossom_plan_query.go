package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (q *Queries) ListBlossomPlans(ctx context.Context, scope string) ([]BlossomPlan, error) {
	args := make([]any, 0, 1)
	where := "1=1"
	if strings.TrimSpace(scope) != "" {
		args = append(args, strings.TrimSpace(scope))
		where = fmt.Sprintf("scope = $%d", len(args))
	}
	query := fmt.Sprintf(`SELECT id, name, scope, storage_quota_bytes, egress_quota_bytes, COALESCE(description, ''), is_default, updated_by, updated_at FROM blossom_plans WHERE %s ORDER BY scope ASC, is_default DESC, updated_at DESC, id ASC`, where)
	rows, err := q.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BlossomPlan, 0, 16)
	for rows.Next() {
		var item BlossomPlan
		if err := rows.Scan(&item.ID, &item.Name, &item.Scope, &item.StorageQuotaBytes, &item.EgressQuotaBytes, &item.Description, &item.IsDefault, &item.UpdatedBy, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetBlossomPlan(ctx context.Context, id string) (BlossomPlan, bool, error) {
	query := `SELECT id, name, scope, storage_quota_bytes, egress_quota_bytes, COALESCE(description, ''), is_default, updated_by, updated_at FROM blossom_plans WHERE id = $1::text LIMIT 1`
	var item BlossomPlan
	err := q.db.QueryRow(ctx, query, strings.TrimSpace(id)).Scan(
		&item.ID, &item.Name, &item.Scope, &item.StorageQuotaBytes, &item.EgressQuotaBytes, &item.Description, &item.IsDefault, &item.UpdatedBy, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BlossomPlan{}, false, nil
		}
		return BlossomPlan{}, false, err
	}
	return item, true, nil
}

func (q *Queries) UpsertBlossomPlan(ctx context.Context, plan BlossomPlan) error {
	pool := DBPool()
	if pool != nil {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin blossom plan tx: %w", err)
		}
		defer tx.Rollback(ctx)

		if plan.IsDefault {
			lockKey := fmt.Sprintf("blossom_plan_default_%s", plan.Scope)
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
				return fmt.Errorf("acquire default plan lock: %w", err)
			}
			if _, err := tx.Exec(ctx, `UPDATE blossom_plans SET is_default = FALSE, updated_at = NOW(), updated_by = $2::text WHERE scope = $1::text AND id != $3::text`, plan.Scope, plan.UpdatedBy, plan.ID); err != nil {
				return fmt.Errorf("reset previous default plans: %w", err)
			}
		}

		_, err = tx.Exec(ctx, `
INSERT INTO blossom_plans (id, name, scope, storage_quota_bytes, egress_quota_bytes, description, is_default, updated_by, updated_at)
VALUES ($1::text, $2::text, $3::text, $4::bigint, $5::bigint, $6::text, $7::boolean, $8::text, NOW())
ON CONFLICT (id) DO UPDATE SET
	name = EXCLUDED.name,
	scope = EXCLUDED.scope,
	storage_quota_bytes = EXCLUDED.storage_quota_bytes,
	egress_quota_bytes = EXCLUDED.egress_quota_bytes,
	description = EXCLUDED.description,
	is_default = EXCLUDED.is_default,
	updated_by = EXCLUDED.updated_by,
	updated_at = NOW()`,
			plan.ID,
			plan.Name,
			plan.Scope,
			nullableInt64(plan.StorageQuotaBytes),
			nullableInt64(plan.EgressQuotaBytes),
			nullableString(plan.Description),
			plan.IsDefault,
			plan.UpdatedBy,
		)
		if err != nil {
			return fmt.Errorf("execute upsert blossom plan: %w", err)
		}

		return tx.Commit(ctx)
	}

	// Fallback for non-pool DBTX executions (e.g. tests)
	if plan.IsDefault {
		if _, err := q.db.Exec(ctx, `UPDATE blossom_plans SET is_default = FALSE, updated_at = NOW(), updated_by = $2::text WHERE scope = $1::text AND id != $3::text`, plan.Scope, plan.UpdatedBy, plan.ID); err != nil {
			return err
		}
	}
	_, err := q.db.Exec(ctx, `
INSERT INTO blossom_plans (id, name, scope, storage_quota_bytes, egress_quota_bytes, description, is_default, updated_by, updated_at)
VALUES ($1::text, $2::text, $3::text, $4::bigint, $5::bigint, $6::text, $7::boolean, $8::text, NOW())
ON CONFLICT (id) DO UPDATE SET
	name = EXCLUDED.name,
	scope = EXCLUDED.scope,
	storage_quota_bytes = EXCLUDED.storage_quota_bytes,
	egress_quota_bytes = EXCLUDED.egress_quota_bytes,
	description = EXCLUDED.description,
	is_default = EXCLUDED.is_default,
	updated_by = EXCLUDED.updated_by,
	updated_at = NOW()`,
		plan.ID,
		plan.Name,
		plan.Scope,
		nullableInt64(plan.StorageQuotaBytes),
		nullableInt64(plan.EgressQuotaBytes),
		nullableString(plan.Description),
		plan.IsDefault,
		plan.UpdatedBy,
	)
	return err
}

func (q *Queries) DeleteBlossomPlan(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, `DELETE FROM blossom_plans WHERE id = $1::text`, id)
	return err
}
