package db

import (
	"context"
	"fmt"
)

// ListBlossomObjectSummaries returns the fields used by the administrative
// library list, using the same filters and ordering as ListBlossomObjects.
func (q *Queries) ListBlossomObjectSummaries(ctx context.Context, filters BlossomObjectFilters, limit, offset int) ([]BlossomObjectRow, int64, error) {
	where, args := buildBlossomObjectsWhere(filters)
	const from = " FROM objects o LEFT JOIN blossom_objects_admin ba ON ba.hash = o.hash LEFT JOIN profiles p ON p.public_key = o.public_key WHERE "
	var total int64
	if err := q.db.QueryRow(ctx, "SELECT COUNT(*)"+from+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), limit, offset)
	statement := fmt.Sprintf(`SELECT o.hash, COALESCE(o.mime_type, ''), COALESCE(o.size, 0), o.created_at,
		COALESCE(ba.thumbnail_hash, ''), COALESCE(ba.review_state, 'ready'), COALESCE(ba.exif_status, 'pending')%s%s
		ORDER BY o.created_at DESC, o.hash DESC LIMIT $%d OFFSET $%d`, from, where, len(listArgs)-1, len(listArgs))
	rows, err := q.db.Query(ctx, statement, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]BlossomObjectRow, 0, limit)
	for rows.Next() {
		var item BlossomObjectRow
		if err := rows.Scan(&item.Hash, &item.MIMEType, &item.Size, &item.CreatedAt, &item.ThumbnailHash, &item.ReviewState, &item.ExifStatus); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
