package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testJob struct {
	Value string `json:"value"`
}

func (testJob) Name() string { return "test.job" }

func TestMarshalEnvelope(t *testing.T) {
	t.Parallel()

	now := time.Unix(1710000000, 0).UTC()
	body, err := MarshalEnvelope(testJob{Value: "ok"}, DispatchConfig{
		Queue:       "admin",
		Priority:    PriorityHigh,
		Timeout:     10 * time.Second,
		MaxAttempts: 4,
		RunAt:       now.Add(30 * time.Second),
	}, now)
	require.NoError(t, err)

	envelope, err := UnmarshalEnvelope(body)
	require.NoError(t, err)
	require.Equal(t, uint8(1), envelope.Version)
	require.Equal(t, "test.job", envelope.Name)
	require.Equal(t, "admin", envelope.Queue)
	require.Equal(t, PriorityHigh, envelope.Priority)
	require.Equal(t, int64(10000), envelope.TimeoutMS)
	require.Equal(t, uint8(4), envelope.MaxAttempts)
	require.NotEmpty(t, envelope.Payload)
}

func TestParseJobID(t *testing.T) {
	t.Parallel()

	id, err := ParseJobID("42")
	require.NoError(t, err)
	require.Equal(t, JobID(42), id)

	_, err = ParseJobID("abc")
	require.Error(t, err)
}
