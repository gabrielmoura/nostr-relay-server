package policies

import "testing"

func TestIsTemporaryRejection(t *testing.T) {
	for _, test := range []struct {
		reason    string
		temporary bool
	}{
		{"error: failed to load group state", true},
		{"error: database unavailable", true},
		{"restricted: event moderation lookup failed", true},
		{"restricted: group access denied", false},
		{"blocked: banned", false},
	} {
		if got := IsTemporaryRejection(test.reason); got != test.temporary {
			t.Errorf("reason=%q temporary=%v want=%v", test.reason, got, test.temporary)
		}
	}
}
