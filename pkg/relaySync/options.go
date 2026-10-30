package relaysync

import (
	"fmt"
	"time"
)

const (
	DefaultMaxEventBytes       = 1 << 20
	DefaultMaxTagCount         = 2_000
	DefaultMaxTagValues        = 100
	DefaultMaxTagValueBytes    = 16 << 10
	DefaultMaxEventsPerBatch   = 500
	DefaultMaxBatchBytes       = 8 << 20
	DefaultMaxFrameBytes       = 4 << 20
	DefaultMaxSessionTTL       = 30 * time.Minute
	DefaultMaxConcurrentRounds = 16
)

// Limits bounds untrusted RelaySync input. Zero values are completed by
// DefaultLimits before validation.
type Limits struct {
	MaxEventBytes       int
	MaxTagCount         int
	MaxTagValues        int
	MaxTagValueBytes    int
	MaxEventsPerBatch   int
	MaxBatchBytes       int
	MaxFrameBytes       int
	MaxSessionTTL       time.Duration
	MaxConcurrentRounds int
}

// DefaultLimits returns conservative limits suitable for a public-facing
// adapter. Hosts may lower these per authorized scope.
func DefaultLimits() Limits {
	return Limits{
		MaxEventBytes:       DefaultMaxEventBytes,
		MaxTagCount:         DefaultMaxTagCount,
		MaxTagValues:        DefaultMaxTagValues,
		MaxTagValueBytes:    DefaultMaxTagValueBytes,
		MaxEventsPerBatch:   DefaultMaxEventsPerBatch,
		MaxBatchBytes:       DefaultMaxBatchBytes,
		MaxFrameBytes:       DefaultMaxFrameBytes,
		MaxSessionTTL:       DefaultMaxSessionTTL,
		MaxConcurrentRounds: DefaultMaxConcurrentRounds,
	}
}

// WithDefaults fills unset values without changing explicit non-zero limits.
func (l Limits) WithDefaults() Limits {
	d := DefaultLimits()
	if l.MaxEventBytes == 0 {
		l.MaxEventBytes = d.MaxEventBytes
	}
	if l.MaxTagCount == 0 {
		l.MaxTagCount = d.MaxTagCount
	}
	if l.MaxTagValues == 0 {
		l.MaxTagValues = d.MaxTagValues
	}
	if l.MaxTagValueBytes == 0 {
		l.MaxTagValueBytes = d.MaxTagValueBytes
	}
	if l.MaxEventsPerBatch == 0 {
		l.MaxEventsPerBatch = d.MaxEventsPerBatch
	}
	if l.MaxBatchBytes == 0 {
		l.MaxBatchBytes = d.MaxBatchBytes
	}
	if l.MaxFrameBytes == 0 {
		l.MaxFrameBytes = d.MaxFrameBytes
	}
	if l.MaxSessionTTL == 0 {
		l.MaxSessionTTL = d.MaxSessionTTL
	}
	if l.MaxConcurrentRounds == 0 {
		l.MaxConcurrentRounds = d.MaxConcurrentRounds
	}
	return l
}

// Validate verifies that all configured limits are positive and internally
// consistent.
func (l Limits) Validate() error {
	if l.MaxEventBytes <= 0 || l.MaxTagCount <= 0 || l.MaxTagValues <= 0 || l.MaxTagValueBytes <= 0 || l.MaxEventsPerBatch <= 0 || l.MaxBatchBytes <= 0 || l.MaxFrameBytes <= 0 || l.MaxSessionTTL <= 0 || l.MaxConcurrentRounds <= 0 {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "limits", Reason: "all limits must be positive"})
	}
	if l.MaxBatchBytes < l.MaxEventBytes {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "max_batch_bytes", Reason: "must be at least max_event_bytes"})
	}
	return nil
}
