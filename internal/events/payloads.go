package events

import "time"

// PresenceEvent describes a connection lifecycle change. Counts are a best
// effort snapshot for the current relay process and are refreshed by normal
// queries after a reconnect.
type PresenceEvent struct {
	Action    string    `json:"action"`
	WSID      string    `json:"wsid,omitempty"`
	Active    int       `json:"active"`
	Authed    int       `json:"authed"`
	ChangedAt time.Time `json:"changedAt"`
}

// JobEvent is emitted after a durable queue transition. Progress fields are
// best-effort and may stay zero for jobs that do not report counters.
type JobEvent struct {
	ID              string    `json:"id"`
	Queue           string    `json:"queue"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	Attempts        int       `json:"attempts"`
	MaxAttempts     int       `json:"maxAttempts"`
	ProgressCurrent int64     `json:"progressCurrent"`
	ProgressTotal   int64     `json:"progressTotal"`
	Stage           string    `json:"stage,omitempty"`
	Error           string    `json:"error,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
