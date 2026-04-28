package jobs

import (
	"fmt"
	"strconv"
	"strings"
)

type Status uint8

const (
	StatusUnknown Status = iota
	StatusQueued
	StatusRunning
	StatusSucceeded
	StatusFailed
	StatusDelayed
	StatusDead
	StatusCanceled
)

func (s Status) String() string {
	switch s {
	case StatusQueued:
		return "queued"
	case StatusRunning:
		return "running"
	case StatusSucceeded:
		return "succeeded"
	case StatusFailed:
		return "failed"
	case StatusDelayed:
		return "delayed"
	case StatusDead:
		return "dead"
	case StatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

func (p Priority) Normalize() Priority {
	switch Priority(strings.ToLower(strings.TrimSpace(string(p)))) {
	case PriorityHigh:
		return PriorityHigh
	case PriorityLow:
		return PriorityLow
	default:
		return PriorityNormal
	}
}

type JobID uint64

func (id JobID) String() string {
	return strconv.FormatUint(uint64(id), 10)
}

func ParseJobID(raw string) (JobID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("job id is required")
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse job id %q: %w", raw, err)
	}

	return JobID(parsed), nil
}
