package _import

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestRejectedEventLineErrorsMapsEveryMatchingJSONLLine(t *testing.T) {
	batch := Batch{
		Items: []*nostr.Event{
			{ID: "event-1"},
			{ID: "invalid"},
			{ID: "event-2"},
			{ID: "invalid"},
		},
		LineNumbers: []int{10, 11, 12, 13},
	}

	errorsByLine := rejectedEventLineErrors(batch, []string{"invalid"})

	require.Len(t, errorsByLine, 2)
	require.Equal(t, 11, errorsByLine[0].LineNumber)
	require.Equal(t, 13, errorsByLine[1].LineNumber)
	require.ErrorContains(t, errorsByLine[0].Err, "invalid")
}
