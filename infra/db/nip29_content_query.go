package db

import "context"

const deleteNIP29GroupContent = `
DELETE FROM event
WHERE kind NOT BETWEEN 9000 AND 9022
  AND kind NOT BETWEEN 39000 AND 39005
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(tags) AS tag
      WHERE tag->>0 = 'h' AND tag->>1 = $1
  )
`

// DeleteNIP29GroupContent removes user-created content while retaining the
// moderation log required to reconstruct the group's canonical state.
func (q *Queries) DeleteNIP29GroupContent(ctx context.Context, groupID string) error {
	_, err := q.db.Exec(ctx, deleteNIP29GroupContent, groupID)
	return err
}
