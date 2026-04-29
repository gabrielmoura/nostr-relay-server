package groups

import (
	"context"
	"strings"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
)

func (m *Manager) validateFilter(ctx context.Context, authed string, filter nostr.Filter) (bool, string) {
	if m == nil || !m.enabled {
		return false, ""
	}
	if !m.shouldValidateFilter(filter) {
		return false, ""
	}

	for _, groupID := range filter.Tags["h"] {
		group, ok, err := m.getGroup(ctx, groupID)
		if err != nil {
			return true, "error: failed to load group policy"
		}
		if !ok {
			continue
		}
		allowed, err := m.canReadGroup(ctx, authed, group, false)
		if err != nil {
			return true, "error: failed to validate group access"
		}
		if !allowed {
			return true, "restricted: not allowed to read this group"
		}
	}

	return false, ""
}

func (m *Manager) validateIncomingEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt.Kind >= nostr.KindSimpleGroupMetadata && evt.Kind <= nostr.KindSimpleGroupRoles {
		return m.reject("metadata_write", "restricted: group metadata events are relay-generated only")
	}

	groupID := groupIDFromEvent(evt)
	if groupID == "" {
		return false, ""
	}

	group, exists, err := m.getGroup(ctx, groupID)
	if err != nil {
		return m.reject("group_lookup_error", "error: failed to load group state")
	}
	if evt.Kind == nostr.KindSimpleGroupCreateGroup {
		return m.validateCreateGroup(ctx, evt, groupID, exists)
	}
	if !exists {
		return m.reject("group_not_found", "invalid: group does not exist")
	}

	if banned, reason, err := m.isBanned(ctx, groupID, evt.PubKey); err != nil {
		return m.reject("ban_lookup_error", "error: failed to validate group ban")
	} else if banned {
		if reason == "" {
			reason = "banned"
		}
		return m.reject("ban", "banned: "+reason)
	}

	switch evt.Kind {
	case nostr.KindSimpleGroupJoinRequest:
		return m.validateJoinRequest(ctx, evt, group)
	case nostr.KindSimpleGroupLeaveRequest:
		return m.validateLeaveRequest(ctx, evt, group)
	case nostr.KindSimpleGroupPutUser,
		nostr.KindSimpleGroupRemoveUser,
		nostr.KindSimpleGroupEditMetadata,
		nostr.KindSimpleGroupDeleteEvent,
		nostr.KindSimpleGroupDeleteGroup,
		nostr.KindSimpleGroupCreateInvite:
		return m.validateModerationEvent(ctx, evt, group)
	default:
		return m.validateGroupContentEvent(ctx, evt, group)
	}
}

func (m *Manager) validateCreateGroup(ctx context.Context, evt *nostr.Event, groupID string, exists bool) (bool, string) {
	if !m.cfg.Create.Enabled {
		return m.reject("create_disabled", "blocked: group creation is disabled")
	}
	if exists {
		return m.reject("duplicate_group", "duplicate: group already exists")
	}
	if !isValidGroupID(groupID) {
		return m.reject("invalid_group_id", "invalid: group id must match [a-z0-9-_]+")
	}
	if m.cfg.Create.MaxGroupsPerPubkey <= 0 {
		return false, ""
	}

	total, err := m.queries.CountNIP29GroupsByCreator(ctx, m.relayScope, evt.PubKey)
	if err != nil {
		return m.reject("create_limit_error", "error: failed to validate group creation limit")
	}
	if total >= m.cfg.Create.MaxGroupsPerPubkey {
		return m.reject("create_limit", "blocked: group creation limit reached")
	}
	return false, ""
}

func (m *Manager) validateJoinRequest(ctx context.Context, evt *nostr.Event, group *dbstore.NIP29Group) (bool, string) {
	member, err := m.isMember(ctx, group.GroupID, evt.PubKey)
	if err != nil {
		return m.reject("member_lookup_error", "error: failed to validate membership")
	}
	if member {
		return m.reject("duplicate_member", "duplicate: already a member")
	}

	code := firstTagValue(evt, "code")
	if code != "" && m.cfg.Invite.Enabled {
		if ok, err := m.validateInvite(ctx, group.GroupID, code); err != nil {
			return m.reject("invite_lookup_error", "error: failed to validate invite code")
		} else if !ok {
			return m.reject("invite_invalid", "restricted: invalid or expired invite code")
		}
	}
	return false, ""
}

func (m *Manager) validateLeaveRequest(ctx context.Context, evt *nostr.Event, group *dbstore.NIP29Group) (bool, string) {
	member, err := m.isMember(ctx, group.GroupID, evt.PubKey)
	if err != nil {
		return m.reject("member_lookup_error", "error: failed to validate membership")
	}
	if !member {
		return m.reject("not_member", "restricted: not a member")
	}
	return false, ""
}

func (m *Manager) validateModerationEvent(ctx context.Context, evt *nostr.Event, group *dbstore.NIP29Group) (bool, string) {
	if m.isStaleModerationEvent(evt) {
		return m.reject("stale_moderation", "blocked: moderation action is too old")
	}
	if !m.hasPermission(ctx, group.GroupID, evt.PubKey, actionName(evt.Kind)) {
		return m.reject("permission", "restricted: insufficient permissions")
	}
	if ok, reason := m.validateTimelineRequirement(ctx, group, evt); !ok {
		return m.reject(reason, rejectionMessage(reason))
	}
	if ok, reason := m.validatePoWRequirement(group, evt, true); !ok {
		return m.reject(reason, rejectionMessage(reason))
	}
	return false, ""
}

func (m *Manager) validateGroupContentEvent(ctx context.Context, evt *nostr.Event, group *dbstore.NIP29Group) (bool, string) {
	if group.Restricted || m.cfg.Admission.RequireMembershipForWrite {
		member, err := m.isMember(ctx, group.GroupID, evt.PubKey)
		if err != nil {
			return m.reject("member_lookup_error", "error: failed to validate membership")
		}
		if !member {
			return m.reject("permission", "restricted: group membership required for publishing")
		}
	}
	if ok, reason := m.validatePoWRequirement(group, evt, false); !ok {
		return m.reject(reason, rejectionMessage(reason))
	}
	if ok, reason := m.validateTimelineRequirement(ctx, group, evt); !ok {
		return m.reject(reason, rejectionMessage(reason))
	}
	return false, ""
}

func (m *Manager) isStaleModerationEvent(evt *nostr.Event) bool {
	if !m.cfg.Moderation.RequireRecentModeration || m.cfg.Moderation.RecentWindowSeconds <= 0 {
		return false
	}
	cutoff := nostr.Now() - nostr.Timestamp(m.cfg.Moderation.RecentWindowSeconds)
	return evt.CreatedAt < cutoff
}

func (m *Manager) validatePoWRequirement(group *dbstore.NIP29Group, evt *nostr.Event, moderation bool) (bool, string) {
	if !m.cfg.PoW.Enabled {
		return true, ""
	}

	required := group.MinPoW
	if required < m.cfg.PoW.DefaultMinDifficulty {
		required = m.cfg.PoW.DefaultMinDifficulty
	}
	if moderation && m.cfg.PoW.ModerationMinDifficulty > required {
		required = m.cfg.PoW.ModerationMinDifficulty
	}
	if required == 0 {
		return true, ""
	}
	if err := nip13.Check(evt.ID, required); err != nil {
		return false, "pow"
	}
	return true, ""
}

func (m *Manager) validateTimelineRequirement(ctx context.Context, group *dbstore.NIP29Group, evt *nostr.Event) (bool, string) {
	if !m.cfg.Timeline.Enabled {
		return true, ""
	}

	requireTag := isModerationKind(evt.Kind) && (m.cfg.Timeline.RequiredOnModeration || group.RequireModerationTimelineRef)
	if !requireTag && m.cfg.Timeline.MinReferences == 0 && group.MinTimelineReferences == 0 {
		return true, ""
	}

	values := allTagValues(evt, "previous")
	required := maxInt(m.cfg.Timeline.MinReferences, group.MinTimelineReferences)
	if requireTag && len(values) < required {
		return false, "timeline_reference"
	}
	if len(values) == 0 {
		return true, ""
	}

	recent, err := m.recentTimelineIDs(ctx, group.GroupID, maxInt(m.cfg.Timeline.RecentWindow, group.TimelineRecentWindow))
	if err != nil {
		return false, "timeline_reference"
	}

	for _, prefix := range values {
		if matchesTimelinePrefix(recent, prefix) {
			continue
		}
		return false, "timeline_reference"
	}

	return true, ""
}

func matchesTimelinePrefix(recent []string, prefix string) bool {
	for _, id := range recent {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

func (m *Manager) reject(reason, message string) (bool, string) {
	metrics.NostrNIP29EventsRejectedTotal.WithLabelValues(reason).Inc()
	return true, message
}

func rejectionMessage(reason string) string {
	switch reason {
	case "pow":
		return "blocked: minimum POW not obtained"
	case "timeline_reference":
		return "restricted: invalid timeline references"
	default:
		return "restricted: event rejected"
	}
}
