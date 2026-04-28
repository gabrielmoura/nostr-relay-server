package jobs

import (
	"fmt"
	"time"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

const envelopeVersion = 1

type Envelope struct {
	Version     uint8           `json:"v"`
	Name        string          `json:"n"`
	Queue       string          `json:"q"`
	Priority    Priority        `json:"p"`
	CreatedAtMS int64           `json:"ca"`
	RunAtMS     int64           `json:"ra,omitempty"`
	TimeoutMS   int64           `json:"to,omitempty"`
	MaxAttempts uint8           `json:"ma"`
	Payload     json.RawMessage `json:"pl"`
	UniqueKey   string          `json:"uk,omitempty"`
	UniqueForMS int64           `json:"uf,omitempty"`
}

func MarshalEnvelope(job Job, cfg DispatchConfig, now time.Time) ([]byte, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("marshal job payload: %w", err)
	}

	envelope := Envelope{
		Version:     envelopeVersion,
		Name:        job.Name(),
		Queue:       cfg.Queue,
		Priority:    cfg.Priority.Normalize(),
		CreatedAtMS: now.UTC().UnixMilli(),
		TimeoutMS:   cfg.Timeout.Milliseconds(),
		MaxAttempts: cfg.MaxAttempts,
		Payload:     payload,
		UniqueKey:   cfg.UniqueKey,
		UniqueForMS: cfg.UniqueFor.Milliseconds(),
	}
	if !cfg.RunAt.IsZero() {
		envelope.RunAtMS = cfg.RunAt.UTC().UnixMilli()
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}

	return body, nil
}

func UnmarshalEnvelope(body []byte) (Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("unmarshal envelope: %w", err)
	}

	if envelope.Name == "" {
		return Envelope{}, fmt.Errorf("envelope missing job name")
	}

	return envelope, nil
}
