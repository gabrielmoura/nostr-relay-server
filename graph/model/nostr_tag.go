package model

import jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"

// UnmarshalJSON customizes the JSON unmarshaling for NostrTag to support
// the standard Nostr tag format (a plain string array like ["p","<pubkey>"]).
// The REST API returns tags as [][]string, but the generated Go struct expects
// an object with a "values" field. This bridge lets jsonx.Unmarshal decode
// the raw array directly into NostrTag.Values.
func (t *NostrTag) UnmarshalJSON(data []byte) error {
	var values []string
	if err := jsonx.Unmarshal(data, &values); err != nil {
		return err
	}
	t.Values = values
	return nil
}

// MarshalJSON customizes the JSON marshaling for NostrTag to emit a plain
// string array, matching the Nostr wire format expected by normalizeRESTValue.
func (t NostrTag) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(t.Values)
}
