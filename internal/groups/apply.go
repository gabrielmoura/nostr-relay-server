package groups

import (
	"context"
	"time"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func (m *Manager) afterStoreEvent(ctx context.Context, evt *nostr.Event) error {
	switch evt.Kind {
	case nostr.KindSimpleGroupCreateGroup:
		return m.applyCreateGroup(ctx, evt)
	case nostr.KindSimpleGroupEditMetadata:
		return m.applyEditMetadata(ctx, evt)
	case nostr.KindSimpleGroupPutUser:
		return m.applyPutUser(ctx, evt)
	case nostr.KindSimpleGroupRemoveUser:
		return m.applyRemoveUser(ctx, evt)
	case nostr.KindSimpleGroupDeleteGroup:
		return m.applyDeleteGroup(ctx, evt)
	case nostr.KindSimpleGroupCreateInvite:
		return m.applyCreateInvite(ctx, evt)
	case nostr.KindSimpleGroupJoinRequest:
		return m.applyJoinRequest(ctx, evt)
	case nostr.KindSimpleGroupLeaveRequest:
		return m.applyLeaveRequest(ctx, evt)
	default:
		m.recordTimelineReference(ctx, evt)
		return nil
	}
}

func (m *Manager) applyCreateGroup(ctx context.Context, evt *nostr.Event) error {
	log.Logger.Debug("applyCreateGroup", zap.Any("event", evt))
	now := time.Now().UTC()
	groupID := groupIDFromEvent(evt)
	group := dbstore.NIP29Group{
		Relay:                m.relayScope,
		GroupID:              groupID,
		Name:                 "",
		Private:              m.cfg.Admission.DefaultPrivate,
		Closed:               m.cfg.Admission.DefaultClosed,
		Restricted:           m.cfg.Admission.DefaultRestricted,
		Hidden:               m.cfg.Admission.DefaultHidden,
		CreatedBy:            evt.PubKey,
		MinPoW:               m.cfg.PoW.DefaultMinDifficulty,
		TimelineRecentWindow: maxInt(m.cfg.Timeline.RecentWindow, 50),
		AllowLatePublication: m.cfg.Admission.AllowLatePublication,
		LastMetadataUpdate:   now,
		LastAdminsUpdate:     now,
		LastMembersUpdate:    now,
		LastRolesUpdate:      now,
	}

	// Clients can specify group metadata tags directly in the 9007 Create Group event
	applyMetadataEdits(&group, evt)

	if m.cfg.Timeline.RequiredOnModeration {
		group.RequireModerationTimelineRef = true
	}
	group.MinTimelineReferences = m.cfg.Timeline.MinReferences

	if err := m.queries.UpsertNIP29Group(ctx, group); err != nil {
		return err
	}
	if err := m.queries.ReplaceNIP29GroupRoles(ctx, m.relayScope, groupID, m.defaultGroupRoleIDs()); err != nil {
		return err
	}
	if err := m.queries.ReplaceNIP29MemberRoles(ctx, m.relayScope, groupID, evt.PubKey, []int32{m.creatorRoleID, m.memberRoleID}); err != nil {
		return err
	}

	metrics.NostrNIP29GroupsCreatedTotal.Inc()
	if err := m.refreshActiveGroupsMetric(ctx); err != nil {
		log.Logger.Debug("failed to refresh nip29 active group metric", zap.Error(err))
	}
	return m.emitStateEvents(ctx, groupID)
}

func (m *Manager) applyEditMetadata(ctx context.Context, evt *nostr.Event) error {
	group, ok, err := m.getGroup(ctx, groupIDFromEvent(evt))
	if err != nil || !ok {
		return err
	}

	applyMetadataEdits(group, evt)
	if err := m.queries.UpsertNIP29Group(ctx, *group); err != nil {
		return err
	}
	m.invalidateGroupCaches(group.GroupID)
	return m.emitStateEvents(ctx, group.GroupID)
}

func (m *Manager) applyPutUser(ctx context.Context, evt *nostr.Event) error {
	groupID := groupIDFromEvent(evt)
	for _, tag := range evt.Tags.GetAll([]string{"p", ""}) {
		pubkey := tag[1]
		roleIDs := m.resolveRoleIDs(tag[2:])
		if err := m.queries.ReplaceNIP29MemberRoles(ctx, m.relayScope, groupID, pubkey, roleIDs); err != nil {
			return err
		}
		if err := m.queries.DeleteNIP29Ban(ctx, m.relayScope, groupID, pubkey); err != nil {
			return err
		}
		m.invalidateMemberCache(groupID, pubkey)
	}
	return m.updateMembershipTimestamps(ctx, groupID, evt)
}

func (m *Manager) applyRemoveUser(ctx context.Context, evt *nostr.Event) error {
	groupID := groupIDFromEvent(evt)
	for _, tag := range evt.Tags.GetAll([]string{"p", ""}) {
		pubkey := tag[1]
		if err := m.queries.RemoveNIP29Member(ctx, m.relayScope, groupID, pubkey); err != nil {
			return err
		}
		m.invalidateMemberCache(groupID, pubkey)
	}
	return m.updateMembershipTimestamps(ctx, groupID, evt)
}

func (m *Manager) applyDeleteGroup(ctx context.Context, evt *nostr.Event) error {
	group, ok, err := m.getGroup(ctx, groupIDFromEvent(evt))
	if err != nil || !ok {
		return err
	}

	now := time.Unix(int64(evt.CreatedAt), 0).UTC()
	group.Name = "[deleted]"
	group.Picture = ""
	group.About = ""
	group.Private = true
	group.Closed = true
	group.Restricted = true
	group.Hidden = true
	group.DeletedAt = &now
	group.LastMetadataUpdate = now
	group.LastAdminsUpdate = now
	group.LastMembersUpdate = now

	if err := m.queries.UpsertNIP29Group(ctx, *group); err != nil {
		return err
	}
	if err := m.queries.ReplaceNIP29GroupRoles(ctx, m.relayScope, group.GroupID, m.defaultGroupRoleIDs()); err != nil {
		return err
	}
	if err := m.emitStateEvents(ctx, group.GroupID); err != nil {
		return err
	}
	return m.refreshActiveGroupsMetric(ctx)
}

func (m *Manager) updateMembershipTimestamps(ctx context.Context, groupID string, evt *nostr.Event) error {
	group, ok, err := m.getGroup(ctx, groupID)
	if err != nil || !ok {
		return err
	}

	updatedAt := time.Unix(int64(evt.CreatedAt), 0).UTC()
	group.LastAdminsUpdate = updatedAt
	group.LastMembersUpdate = updatedAt
	if err := m.queries.UpsertNIP29Group(ctx, *group); err != nil {
		return err
	}
	return m.emitStateEvents(ctx, groupID)
}

func (m *Manager) emitStateEvents(ctx context.Context, groupID string) error {
	group, ok, err := m.getGroup(ctx, groupID)
	if err != nil || !ok {
		return err
	}

	events, err := m.stateEvents(ctx, group)
	if err != nil {
		return err
	}
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if err := evt.Sign(m.relayPrivKey); err != nil {
			return err
		}
		if err := m.persistRelayEvent(ctx, evt); err != nil {
			return err
		}
		listener.NotifyListeners(evt)
	}

	m.invalidateGroupCaches(groupID)
	return nil
}

func (m *Manager) stateEvents(ctx context.Context, group *dbstore.NIP29Group) ([]*nostr.Event, error) {
	admins, members, roles, err := m.buildStateEvents(ctx, group)
	if err != nil {
		return nil, err
	}
	metadata := &nostr.Event{
		PubKey:    m.relayPubKey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindSimpleGroupMetadata,
		Tags:      buildMetadataTags(group),
		Content:   "",
	}
	return []*nostr.Event{metadata, admins, members, roles}, nil
}
