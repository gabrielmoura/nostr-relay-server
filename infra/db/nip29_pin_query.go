package db

import "context"

const replaceNIP29PinsDelete = `
DELETE FROM nip29_group_pins
WHERE relay = $1 AND group_id = $2
`

const insertNIP29Pin = `
INSERT INTO nip29_group_pins (relay, group_id, position, reference_type, reference_value)
VALUES ($1, $2, $3, $4, $5)
`

func (q *Queries) ReplaceNIP29Pins(ctx context.Context, relay, groupID string, pins []NIP29Pin) error {
	if _, err := q.db.Exec(ctx, replaceNIP29PinsDelete, relay, groupID); err != nil {
		return err
	}
	for position, pin := range pins {
		if _, err := q.db.Exec(ctx, insertNIP29Pin, relay, groupID, position, pin.ReferenceType, pin.ReferenceValue); err != nil {
			return err
		}
	}
	return nil
}

const listNIP29Pins = `
SELECT reference_type, reference_value
FROM nip29_group_pins
WHERE relay = $1 AND group_id = $2
ORDER BY position
`

func (q *Queries) ListNIP29Pins(ctx context.Context, relay, groupID string) ([]NIP29Pin, error) {
	rows, err := q.db.Query(ctx, listNIP29Pins, relay, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pins := make([]NIP29Pin, 0, 4)
	for rows.Next() {
		var pin NIP29Pin
		if err := rows.Scan(&pin.ReferenceType, &pin.ReferenceValue); err != nil {
			return nil, err
		}
		pins = append(pins, pin)
	}
	return pins, rows.Err()
}
