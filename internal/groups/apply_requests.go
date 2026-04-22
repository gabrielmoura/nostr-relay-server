package groups

import (
	"context"
	"errors"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
)

func (m *Manager) applyCreateInvite(ctx context.Context, evt *nostr.Event) error {
	if !m.cfg.Invite.Enabled {
		return nil
	}

	code := firstTagValue(evt, "code")
	if code == "" {
		return nil
	}

	invite := dbstore.NIP29Invite{
		Relay:     m.relayScope,
		GroupID:   groupIDFromEvent(evt),
		Code:      code,
		CreatedBy: evt.PubKey,
		MaxUses:   maxInt(1, m.cfg.Invite.DefaultMaxUses),
		ExpiresAt: inviteExpiry(m.cfg.Invite.DefaultTTLSeconds),
	}
	if err := m.queries.UpsertNIP29Invite(ctx, invite); err != nil {
		return err
	}
	metrics.NostrNIP29InvitesGeneratedTotal.Inc()
	m.invalidateInviteCache(invite.GroupID, invite.Code)
	return nil
}

func (m *Manager) applyJoinRequest(ctx context.Context, evt *nostr.Event) error {
	groupID := groupIDFromEvent(evt)
	if code := firstTagValue(evt, "code"); code != "" && m.cfg.Invite.Enabled {
		used, err := m.consumeInvite(ctx, groupID, code)
		if err != nil {
			return err
		}
		if used {
			metrics.NostrNIP29InvitesConsumedTotal.Inc()
		}
	}

	internal := &nostr.Event{
		PubKey:    m.relayPubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindSimpleGroupPutUser,
		Tags:      nostr.Tags{{"h", groupID}, {"p", evt.PubKey, "member"}},
		Content:   "join accepted",
	}
	return m.saveInternalEventAndApply(ctx, internal)
}

func (m *Manager) applyLeaveRequest(ctx context.Context, evt *nostr.Event) error {
	internal := &nostr.Event{
		PubKey:    m.relayPubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindSimpleGroupRemoveUser,
		Tags:      nostr.Tags{{"h", groupIDFromEvent(evt)}, {"p", evt.PubKey}},
		Content:   "member left",
	}
	return m.saveInternalEventAndApply(ctx, internal)
}

func (m *Manager) saveInternalEventAndApply(ctx context.Context, evt *nostr.Event) error {
	if err := evt.Sign(m.relayPrivKey); err != nil {
		return err
	}
	if err := m.persistRelayEvent(ctx, evt); err != nil {
		return err
	}
	listener.NotifyListeners(evt)
	return AfterStoreEvent(ctx, evt)
}

func (m *Manager) persistRelayEvent(ctx context.Context, evt *nostr.Event) error {
	if err := m.deletePreviousReplaceableEvents(ctx, evt); err != nil {
		return err
	}
	if err := m.queries.InsertEvent(ctx, evt); err != nil && !errors.Is(err, dbstore.ErrDupEvent) {
		return err
	}
	return nil
}

func (m *Manager) deletePreviousReplaceableEvents(ctx context.Context, evt *nostr.Event) error {
	if !nostr.IsReplaceableKind(evt.Kind) && !nostr.IsAddressableKind(evt.Kind) {
		return nil
	}

	filter := nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}, Limit: 20}
	if nostr.IsAddressableKind(evt.Kind) {
		if d := firstTagValue(evt, "d"); d != "" {
			filter.Tags = nostr.TagMap{"d": []string{d}}
		}
	}

	ch, err := m.queries.QueryEventsChan(ctx, filter)
	if err != nil {
		return err
	}
	for previous := range ch {
		if previous.ID == evt.ID {
			continue
		}
		if err := m.queries.DeleteEvent(ctx, previous.ID, evt.ID); err != nil {
			return err
		}
	}
	return nil
}
