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

const (
	kindSimpleGroupUpdatePinList = 9010
	kindSimpleGroupPinnedEvents  = 39005
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
	// Filtering must happen for every read while NIP-29 is enabled. A generic
	// filter can return regular events that carry an h tag for a private group;
	// deciding from the requested filter alone would expose them to anonymous
	// clients.
	return true
}

func (m *Manager) isRelevantEvent(evt *nostr.Event) bool {
	if evt == nil {
		return false
	}
	if isNIP29MetadataKind(evt.Kind) {
		return true
	}
	return isNIP29ScopedWriteKind(evt.Kind) || firstTagValue(evt, "h") != ""
}

func (m *Manager) forwardAllowedEvents(ctx context.Context, authed string, results <-chan *nostr.Event, out chan<- *nostr.Event) {
	defer close(out)
	reader := newReadAuthorizer(m, authed)
	for evt := range results {
		allowed, err := reader.canReadEvent(ctx, evt)
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

type groupReadResult struct {
	group  *dbstore.NIP29Group
	exists bool
	err    error
}

type memberReadResult struct {
	allowed bool
	err     error
}

type readAuthorizer struct {
	lookupGroup  func(context.Context, string) (*dbstore.NIP29Group, bool, error)
	lookupMember func(context.Context, string, string) (bool, error)
	authed       string
	groups       map[string]groupReadResult
	members      map[string]memberReadResult
}

func newReadAuthorizer(m *Manager, authed string) *readAuthorizer {
	// Authorization must not rely on cache-aside entries: an older read can
	// repopulate the cache after a privacy change or membership revocation.
	// The authorizer still memoizes lookups for the duration of this read.
	return &readAuthorizer{
		lookupGroup: func(ctx context.Context, groupID string) (*dbstore.NIP29Group, bool, error) {
			return m.queries.GetNIP29Group(ctx, m.relayScope, groupID)
		},
		lookupMember: func(ctx context.Context, groupID, pubkey string) (bool, error) {
			if pubkey == "" {
				return false, nil
			}
			return m.queries.IsNIP29Member(ctx, m.relayScope, groupID, pubkey)
		},
		authed: authed, groups: make(map[string]groupReadResult), members: make(map[string]memberReadResult),
	}
}

func (r *readAuthorizer) canReadEvent(ctx context.Context, evt *nostr.Event) (bool, error) {
	groupID, metadata := readEventGroup(evt)
	if groupID == "" {
		return !metadata && !isNIP29ScopedWriteKind(evt.Kind), nil
	}
	groupResult, cached := r.groups[groupID]
	if !cached {
		groupResult.group, groupResult.exists, groupResult.err = r.lookupGroup(ctx, groupID)
		r.groups[groupID] = groupResult
	}
	if groupResult.err != nil {
		return false, groupResult.err
	}
	if !groupResult.exists {
		// A missing projection must not expose events that are unmistakably
		// NIP-29 operations. Generic events with an h tag remain readable: that
		// tag is not exclusively owned by NIP-29 and cannot be classified safely
		// without a durable group-scope marker at ingestion time.
		return !isNIP29ScopedWriteKind(evt.Kind), nil
	}
	if !groupResult.group.Private && !(groupResult.group.Hidden && metadata) {
		return true, nil
	}
	member, cached := r.members[groupID]
	if !cached {
		member.allowed, member.err = r.lookupMember(ctx, groupID, r.authed)
		r.members[groupID] = member
	}
	return member.allowed, member.err
}

func readEventGroup(evt *nostr.Event) (string, bool) {
	if evt == nil {
		return "", false
	}
	metadata := isNIP29MetadataKind(evt.Kind)
	groupID := firstTagValue(evt, "h")
	if metadata {
		groupID = firstTagValue(evt, "d")
	}
	return groupID, metadata
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
	case kindSimpleGroupUpdatePinList:
		return "update-pin-list"
	default:
		return strconv.Itoa(kind)
	}
}

func isModerationKind(kind int) bool {
	return kind >= 9000 && kind <= 9020
}

func isNIP29MetadataKind(kind int) bool {
	return kind >= nostr.KindSimpleGroupMetadata && kind <= kindSimpleGroupPinnedEvents
}

func isNIP29ScopedWriteKind(kind int) bool {
	return isNIP29MetadataKind(kind) || (kind >= 9000 && kind <= 9022)
}

func requiresGroupTag(kind int) bool {
	return kind >= 9000 && kind <= 9022
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
