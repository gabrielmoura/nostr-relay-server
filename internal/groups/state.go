package groups

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func (m *Manager) bootstrapRoles(ctx context.Context) error {
	memberRole := configRoleMember()
	if err := m.registerRole(ctx, memberRole); err != nil {
		return err
	}
	m.memberRoleID = m.roleIDs[memberRole.Name]

	for _, role := range m.cfg.DefaultRoles {
		if role.Name == "" {
			continue
		}
		if err := m.registerRole(ctx, role); err != nil {
			return err
		}
	}

	creatorRoleName := m.cfg.GroupCreatorRole
	if creatorRoleName == "" {
		creatorRoleName = "admin"
	}
	creatorRoleID, ok := m.roleIDs[creatorRoleName]
	if !ok {
		return errMissingCreatorRole(creatorRoleName)
	}
	m.creatorRoleID = creatorRoleID
	return nil
}

func (m *Manager) registerRole(ctx context.Context, role config.NIP29RoleConfig) error {
	m.roleConfigs[role.Name] = role
	roleID, err := m.queries.EnsureNIP29Role(ctx, role.Name, role.Description)
	if err != nil {
		return errEnsureRole(role.Name, err)
	}
	m.roleIDs[role.Name] = roleID
	return nil
}

func (m *Manager) shouldValidateFilter(filter nostr.Filter) bool {
	return len(filter.Tags["h"]) > 0
}

func (m *Manager) shouldFilterQueryResults(filter nostr.Filter) bool {
	for _, kind := range filter.Kinds {
		if isNIP29MetadataKind(kind) {
			return true
		}
	}
	return len(filter.Tags["h"]) > 0 || len(filter.Tags["d"]) > 0 || len(filter.IDs) > 0 || len(filter.Tags["e"]) > 0 || len(filter.Tags["a"]) > 0
}

func (m *Manager) isRelevantEvent(evt *nostr.Event) bool {
	if evt == nil {
		return false
	}
	if isNIP29MetadataKind(evt.Kind) {
		return true
	}
	if !isNIP29ScopedWriteKind(evt.Kind) {
		return false
	}
	return groupIDFromEvent(evt) != ""
}

func (m *Manager) forwardAllowedEvents(ctx context.Context, authed string, results <-chan *nostr.Event, out chan<- *nostr.Event) {
	defer close(out)
	for evt := range results {
		allowed, err := m.canReadEvent(ctx, authed, evt)
		if err != nil {
			log.Logger.Debug("nip29 read filter error", zap.Error(err), zap.String("event_id", evt.ID))
			continue
		}
		if !allowed {
			continue
		}
		select {
		case out <- evt:
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) canReadEvent(ctx context.Context, authed string, evt *nostr.Event) (bool, error) {
	groupID := groupIDFromEvent(evt)
	if groupID == "" {
		return true, nil
	}
	group, ok, err := m.getGroup(ctx, groupID)
	if err != nil || !ok {
		return true, err
	}
	isMetadata := evt.Kind >= nostr.KindSimpleGroupMetadata && evt.Kind <= nostr.KindSimpleGroupRoles
	return m.canReadGroup(ctx, authed, group, isMetadata)
}

func (m *Manager) canReadGroup(ctx context.Context, authed string, group *dbstore.NIP29Group, metadata bool) (bool, error) {
	if group.Hidden && metadata {
		return m.isMember(ctx, group.GroupID, authed)
	}
	if group.Private {
		return m.isMember(ctx, group.GroupID, authed)
	}
	return true, nil
}

func (m *Manager) hasPermission(ctx context.Context, groupID, pubkey, action string) bool {
	if pubkey == m.relayPubKey {
		return true
	}
	roles, err := m.queries.GetNIP29MemberRoleNames(ctx, m.relayScope, groupID, pubkey)
	if err != nil {
		return false
	}
	for _, roleName := range roles {
		cfg, ok := m.roleConfigs[roleName]
		if !ok {
			continue
		}
		if slices.Contains(cfg.Permissions, action) || slices.Contains(cfg.Permissions, "*") {
			return true
		}
	}
	return false
}

func (m *Manager) hasAdminRole(roleNames []string) bool {
	for _, roleName := range roleNames {
		cfg, ok := m.roleConfigs[roleName]
		if ok && len(cfg.Permissions) > 0 {
			return true
		}
	}
	return false
}

func (m *Manager) defaultGroupRoleIDs() []int32 {
	roleIDs := make([]int32, 0, len(m.roleIDs))
	for _, roleID := range m.roleIDs {
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

func (m *Manager) getGroup(ctx context.Context, groupID string) (*dbstore.NIP29Group, bool, error) {
	if groupID == "" {
		return nil, false, nil
	}
	if group, ok := m.getGroupFromCache(groupID); ok {
		return group, true, nil
	}
	group, ok, err := m.queries.GetNIP29Group(ctx, m.relayScope, groupID)
	if err != nil || !ok {
		return group, ok, err
	}
	m.storeGroupCache(group)
	return group, true, nil
}

func (m *Manager) isMember(ctx context.Context, groupID, pubkey string) (bool, error) {
	if pubkey == "" {
		return false, nil
	}
	if cached, ok := m.getMembershipCache(groupID, pubkey); ok {
		return cached, nil
	}
	exists, err := m.queries.IsNIP29Member(ctx, m.relayScope, groupID, pubkey)
	if err != nil {
		return false, err
	}
	m.storeMembershipCache(groupID, pubkey, exists)
	return exists, nil
}

func (m *Manager) isBanned(ctx context.Context, groupID, pubkey string) (bool, string, error) {
	if pubkey == "" {
		return false, "", nil
	}
	if cached, reason, ok := m.getBanCache(groupID, pubkey); ok {
		return cached, reason, nil
	}
	reason, exists, err := m.queries.GetNIP29Ban(ctx, m.relayScope, groupID, pubkey)
	if err != nil {
		return false, "", err
	}
	m.storeBanCache(groupID, pubkey, exists, reason)
	return exists, reason, nil
}

func (m *Manager) validateInvite(ctx context.Context, groupID, code string) (bool, error) {
	if invite, ok := m.getInviteCache(groupID, code); ok {
		return inviteValid(invite), nil
	}
	invite, ok, err := m.queries.GetNIP29Invite(ctx, m.relayScope, groupID, code)
	if err != nil || !ok {
		return false, err
	}
	m.storeInviteCache(invite)
	return inviteValid(invite), nil
}

func (m *Manager) consumeInvite(ctx context.Context, groupID, code string) (bool, error) {
	ok, err := m.queries.ConsumeNIP29Invite(ctx, m.relayScope, groupID, code)
	if ok {
		m.invalidateInviteCache(groupID, code)
	}
	return ok, err
}

func (m *Manager) recentTimelineIDs(ctx context.Context, groupID string, window int) ([]string, error) {
	if window <= 0 {
		window = 50
	}
	if ids := m.getTimelineFromCache(groupID, window); len(ids) > 0 {
		return ids, nil
	}
	filter := nostr.Filter{Tags: nostr.TagMap{"h": []string{groupID}}, Limit: window}
	events, err := m.queries.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(events))
	for _, evt := range events {
		ids = append(ids, evt.ID)
	}
	return ids, nil
}

func (m *Manager) refreshActiveGroupsMetric(ctx context.Context) error {
	total, err := m.queries.CountNIP29ActiveGroups(ctx, m.relayScope)
	if err != nil {
		return err
	}
	metrics.NostrNIP29GroupsActive.Set(float64(total))
	return nil
}

func actionName(kind int) string {
	switch kind {
	case nostr.KindSimpleGroupCreateGroup:
		return "create-group"
	case nostr.KindSimpleGroupPutUser:
		return "put-user"
	case nostr.KindSimpleGroupRemoveUser:
		return "remove-user"
	case nostr.KindSimpleGroupEditMetadata:
		return "edit-metadata"
	case nostr.KindSimpleGroupDeleteEvent:
		return "delete-event"
	case nostr.KindSimpleGroupDeleteGroup:
		return "delete-group"
	case nostr.KindSimpleGroupCreateInvite:
		return "create-invite"
	default:
		return strconv.Itoa(kind)
	}
}

func isModerationKind(kind int) bool {
	return kind >= 9000 && kind <= 9020
}

func isNIP29MetadataKind(kind int) bool {
	return kind >= nostr.KindSimpleGroupMetadata && kind <= nostr.KindSimpleGroupRoles
}

func isNIP29ScopedWriteKind(kind int) bool {
	return isNIP29MetadataKind(kind) || (kind >= 9000 && kind <= 9022)
}

func inviteValid(invite *dbstore.NIP29Invite) bool {
	if invite == nil {
		return false
	}
	if invite.RevokedAt != nil {
		return false
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now().UTC()) {
		return false
	}
	return invite.Uses < invite.MaxUses
}
