package down

import (
	"testing"
	"time"

	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestSnapshotToJob(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(QueueDownloadJob{
		Request:    JobRequest{Relays: []string{"wss://relay.damus.io"}, Timeout: 30},
		Filter:     nostr.Filter{Kinds: []int{1}},
		FilterJSON: `{"kinds":[1]}`,
	})
	require.NoError(t, err)
	result, err := json.Marshal(QueueDownloadResult{
		Summary: DownloadSummary{EventsReceived: 10, InsertedEvents: 8},
	})
	require.NoError(t, err)

	snapshot := jobcore.Snapshot{
		ID:        jobcore.JobID(12),
		Queue:     "admin",
		Name:      queueJobName,
		Status:    jobcore.StatusSucceeded,
		CreatedAt: time.Unix(1710000000, 0).UTC(),
		Payload:   payload,
		Result:    result,
		LastError: "",
	}

	job, err := snapshotToJob(snapshot)
	require.NoError(t, err)
	require.Equal(t, "dl_12", job.ID)
	require.Equal(t, JobCompleted, job.Status)
	require.Equal(t, []string{"wss://relay.damus.io"}, job.Relays)
	require.Equal(t, 10, job.Summary.EventsReceived)
	require.Equal(t, 8, job.Summary.InsertedEvents)
}
