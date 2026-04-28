package redisqueue

import (
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	t.Parallel()

	keys := NewKeys("admin")
	require.Equal(t, "rq:{admin}:seq", keys.Seq())
	require.Equal(t, "rq:{admin}:stream:high", keys.Stream(jobs.PriorityHigh))
	require.Equal(t, "rq:{admin}:body:7", keys.Body(jobs.JobID(7)))
	require.Equal(t, "rq:{admin}:result:7", keys.Result(jobs.JobID(7)))
	require.Equal(t, "rq:{admin}:meta:7", keys.Meta(jobs.JobID(7)))
	require.Equal(t, "rq:{admin}:metrics:202601020304", keys.MetricsBucket(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)))
}
