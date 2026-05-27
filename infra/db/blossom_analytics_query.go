package db

import "context"

func (q *Queries) GetBlossomObjectCountsByMIME(ctx context.Context) ([]BlossomCountByValue, error) {
	rows, err := q.db.Query(ctx, `SELECT COALESCE(NULLIF(mime_type, ''), 'unknown') AS name, COUNT(*) AS count FROM objects GROUP BY COALESCE(NULLIF(mime_type, ''), 'unknown') ORDER BY count DESC, name ASC LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BlossomCountByValue, 0, 8)
	for rows.Next() {
		var item BlossomCountByValue
		if err := rows.Scan(&item.Name, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetBlossomObjectCountsByReviewState(ctx context.Context) ([]BlossomCountByValue, error) {
	rows, err := q.db.Query(ctx, `SELECT COALESCE(review_state, 'ready') AS name, COUNT(*) AS count FROM blossom_objects_admin GROUP BY COALESCE(review_state, 'ready') ORDER BY count DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BlossomCountByValue, 0, 8)
	for rows.Next() {
		var item BlossomCountByValue
		if err := rows.Scan(&item.Name, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
