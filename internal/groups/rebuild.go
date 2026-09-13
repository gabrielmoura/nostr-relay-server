package groups

import (
	"context"
	"fmt"
)

func (m *Manager) rebuildState(ctx context.Context) error {
	events, err := m.queries.ListNIP29ModerationEvents(ctx)
	if err != nil {
		return fmt.Errorf("list NIP-29 moderation events: %w", err)
	}
	if err := m.queries.ResetNIP29Projection(ctx, m.relayScope); err != nil {
		return fmt.Errorf("reset NIP-29 projection: %w", err)
	}

	m.rebuilding = true
	defer func() { m.rebuilding = false }()
	for _, evt := range events {
		// Deletion events affect event storage, not the reconstructable group state.
		if evt.Kind == 9005 {
			continue
		}
		if err := m.afterStoreEvent(ctx, evt); err != nil {
			return fmt.Errorf("replay NIP-29 event %s: %w", evt.ID, err)
		}
	}

	return nil
}

func (m *Manager) emitRebuiltStateEvents(ctx context.Context) error {
	groupIDs, err := m.queries.ListNIP29GroupIDs(ctx, m.relayScope)
	if err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		if err := m.emitStateEvents(ctx, groupID); err != nil {
			return err
		}
	}
	return nil
}
