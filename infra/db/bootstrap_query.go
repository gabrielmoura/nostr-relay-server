package db

import "context"

const countEventsByTagValueSQL = `SELECT COUNT(*) FROM event WHERE $1 = ANY(tagvalues)`

func (q *Queries) CountEventsByTagValue(ctx context.Context, tagValue string) (int64, error) {
	var count int64
	err := q.db.QueryRow(ctx, countEventsByTagValueSQL, tagValue).Scan(&count)
	return count, err
}
