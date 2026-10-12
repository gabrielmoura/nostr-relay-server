package relaysync

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidEvent        = errors.New("relaysync: invalid event")
	ErrInvalidIdentifier   = errors.New("relaysync: invalid identifier")
	ErrInvalidLimits       = errors.New("relaysync: invalid limits")
	ErrNotFound            = errors.New("relaysync: not found")
	ErrConflict            = errors.New("relaysync: conflict")
	ErrSnapshotExpired     = errors.New("relaysync: snapshot expired")
	ErrSessionExpired      = errors.New("relaysync: session expired")
	ErrJournalGap          = errors.New("relaysync: journal gap")
	ErrJournalEpochChanged = errors.New("relaysync: journal epoch changed")
	ErrReconcileRequired   = errors.New("relaysync: reconcile required")
	ErrFlowControl         = errors.New("relaysync: flow control paused")
	ErrHeartbeatTimeout    = errors.New("relaysync: heartbeat timeout")
	ErrUnauthorized        = errors.New("relaysync: unauthorized")
)

// ValidationError identifies an invalid input field without exposing its value.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("relaysync: invalid %s: %s", e.Field, e.Reason)
}

func invalidEvent(field, reason string) error {
	return fmt.Errorf("%w: %w", ErrInvalidEvent, &ValidationError{Field: field, Reason: reason})
}
