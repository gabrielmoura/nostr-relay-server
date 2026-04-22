package groups

import (
	"context"
	"slices"
	"time"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

func (m *Manager) buildStateEvents(ctx context.Context, group *dbstore.NIP29Group) (*nostr.Event, *nostr.Event, *nostr.Event, error) {
	memberRoles, err := m.queries.ListNIP29MemberRoles(ctx, m.relayScope, group.GroupID)
	if err != nil {
		return nil, nil, nil, err
	}
	groupRoles, err := m.queries.ListNIP29GroupRoles(ctx, m.relayScope, group.GroupID)
	if err != nil {
		return nil, nil, nil, err
	}

	adminsTags, membersTags := memberStateTags(group.GroupID, memberRoles, m.hasAdminRole)
	rolesTags := roleStateTags(group.GroupID, groupRoles)

	admins := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: nostr.KindSimpleGroupAdmins, Tags: adminsTags}
	members := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: nostr.KindSimpleGroupMembers, Tags: membersTags}
	roles := &nostr.Event{PubKey: m.relayPubKey, CreatedAt: nostr.Now(), Kind: nostr.KindSimpleGroupRoles, Tags: rolesTags}

	if !m.cfg.Advanced.EmitMemberListEvents {
		members = nil
	}
	if !m.cfg.Advanced.EmitRoleEvents {
		roles = nil
	}
	return admins, members, roles, nil
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
	group.Private = tagExists(evt, "private")
	group.Closed = tagExists(evt, "closed")
	group.Restricted = tagExists(evt, "restricted")
	group.Hidden = tagExists(evt, "hidden")
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
	roleIDs := []int32{m.memberRoleID}
	for _, roleName := range roleNames {
		if roleID, ok := m.roleIDs[roleName]; ok && !slices.Contains(roleIDs, roleID) {
			roleIDs = append(roleIDs, roleID)
		}
	}
	return roleIDs
}
