package groups

import (
	"context"
	"slices"
	"time"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

func (m *Manager) buildStateEvents(ctx context.Context, group *dbstore.NIP29Group) (*nostr.Event, *nostr.Event, *nostr.Event, *nostr.Event, error) {
	memberRoles, err := m.queries.ListNIP29MemberRoles(ctx, m.relayScope, group.GroupID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	groupRoles, err := m.queries.ListNIP29GroupRoles(ctx, m.relayScope, group.GroupID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	pins, err := m.queries.ListNIP29Pins(ctx, m.relayScope, group.GroupID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	adminsTags, membersTags := memberStateTags(group.GroupID, memberRoles, m.hasAdminRole)
	rolesTags := roleStateTags(group.GroupID, groupRoles)

	admins := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: nostr.KindSimpleGroupAdmins, Tags: adminsTags}
	members := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: nostr.KindSimpleGroupMembers, Tags: membersTags}
	roles := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: nostr.KindSimpleGroupRoles, Tags: rolesTags}
	pinnedTags := nostr.Tags{{"d", group.GroupID}}
	for _, pin := range pins {
		pinnedTags = append(pinnedTags, nostr.Tag{pin.ReferenceType, pin.ReferenceValue})
	}
	pinned := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: kindSimpleGroupPinnedEvents, Tags: pinnedTags}

	if !m.cfg.Advanced.EmitMemberListEvents {
		members = nil
	}
	if !m.cfg.Advanced.EmitRoleEvents {
		roles = nil
	}
	return admins, members, roles, pinned, nil
}

func applyMetadataEdits(group *dbstore.NIP29Group, evt *nostr.Event) {
	if value := firstTagValue(evt, "name"); value != "" {
		group.Name = value
	}
	if value := firstTagValue(evt, "picture"); value != "" {
		group.Picture = value
	}
	if value := firstTagValue(evt, "about"); value != "" {
		group.About = value
	}
	if visibility := firstTagValue(evt, "visibility"); visibility != "" {
		private, closed, restricted, hidden, ok := visibilityFlags(visibility)
		if ok {
			group.Private = private
			group.Closed = closed
			group.Restricted = restricted
			group.Hidden = hidden
		}
	} else {
		group.Private = tagExists(evt, "private")
		group.Closed = tagExists(evt, "closed")
		group.Restricted = tagExists(evt, "restricted")
		group.Hidden = tagExists(evt, "hidden")
	}
	if topics := allTagValues(evt, "t"); len(topics) > 0 {
		group.Topics = topics
	}
	if geohashes := allTagValues(evt, "g"); len(geohashes) > 0 {
		group.Geohashes = geohashes
	}
	group.LastMetadataUpdate = time.Unix(int64(evt.CreatedAt), 0).UTC()
}

func inviteExpiry(ttlSeconds int) *time.Time {
	if ttlSeconds <= 0 {
		return nil
	}
	t := time.Now().UTC().Add(time.Duration(ttlSeconds) * time.Second)
	return &t
}

func (m *Manager) resolveRoleIDs(roleNames []string) []int32 {
	roleIDs := make([]int32, 0, len(roleNames)+1)
	for _, roleName := range roleNames {
		if roleID, ok := m.roleIDs[roleName]; ok && !slices.Contains(roleIDs, roleID) {
			roleIDs = append(roleIDs, roleID)
		}
	}
	if len(roleIDs) == 0 {
		roleIDs = append(roleIDs, m.memberRoleID)
	}
	return roleIDs
}
