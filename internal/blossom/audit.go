package blossom

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	storedb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

func RecordAudit(ctx context.Context, actorPubkey, action, targetType, targetID string, payload map[string]string) error {
	if storedb.DbQueries == nil {
		return fmt.Errorf("database queries not initialized")
	}

	content, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal blossom audit payload: %w", err)
	}

	record := dbmodel.BlossomAuditRecord{
		ActorPubkey: normalizeActorPubkey(actorPubkey),
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Payload:     content,
		CreatedAt:   time.Now().UTC(),
	}

	if evt, evtErr := buildAuditEvent(record); evtErr == nil {
		if signErr := evt.Sign(config.Cfg.RelayInformation.PrivKey); signErr == nil {
			record.NostrEventID = evt.ID
			_ = storedb.DbQueries.InsertEvent(ctx, evt)
		}
	}

	if err := storedb.DbQueries.InsertBlossomAuditLog(ctx, record); err != nil {
		return fmt.Errorf("insert blossom audit log: %w", err)
	}

	return nil
}

func buildAuditEvent(record dbmodel.BlossomAuditRecord) (*nostr.Event, error) {
	privKey := strings.TrimSpace(config.Cfg.RelayInformation.PrivKey)
	if privKey == "" {
		return nil, fmt.Errorf("relay_information.priv_key is required")
	}
	return &nostr.Event{
		CreatedAt: nostr.Timestamp(record.CreatedAt.Unix()),
		Kind:      24242,
		Content:   string(record.Payload),
		Tags: nostr.Tags{
			{"t", "blossom"},
			{"action", record.Action},
			{"target_type", record.TargetType},
			{"target_id", record.TargetID},
			{"actor", record.ActorPubkey},
		},
	}, nil
}

func normalizeActorPubkey(actorPubkey string) string {
	actorPubkey = strings.TrimSpace(actorPubkey)
	if actorPubkey != "" {
		return actorPubkey
	}
	if strings.TrimSpace(config.Cfg.AdminPubKey) != "" {
		return config.Cfg.AdminPubKey
	}
	return config.Cfg.RelayInformation.PubKey
}
