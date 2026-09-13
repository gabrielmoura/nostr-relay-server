package groups

import (
	"context"
	"slices"

	"github.com/nbd-wtf/go-nostr"
)

func (m *Manager) validateSubgroupMetadata(ctx context.Context, groupID, pubkey string, evt *nostr.Event) (bool, string) {
	parents := allTagValues(evt, "parent")
	if len(parents) > 1 {
		return false, "subgroup_parent"
	}
	parentGroupID := firstTagValue(evt, "parent")
	if parentGroupID != "" {
		parent, exists, err := m.getGroup(ctx, parentGroupID)
		if err != nil || !exists || parent.DeletedAt != nil {
			return false, "subgroup_parent"
		}
		if !m.isGroupAdmin(ctx, parent.GroupID, pubkey) || m.wouldCreateSubgroupCycle(ctx, groupID, parent.GroupID) {
			return false, "subgroup_parent"
		}
	}

	children, err := m.queries.ListNIP29Children(ctx, m.relayScope, groupID)
	if err != nil {
		return false, "subgroup_children"
	}
	requested := allTagValues(evt, "child")
	if !sameGroupIDs(children, requested) {
		return false, "subgroup_children"
	}
	return true, ""
}

func (m *Manager) applySubgroupMetadata(ctx context.Context, groupID string, evt *nostr.Event) error {
	if err := m.queries.SetNIP29Parent(ctx, m.relayScope, groupID, firstTagValue(evt, "parent")); err != nil {
		return err
	}
	children := allTagValues(evt, "child")
	if len(children) == 0 {
		return nil
	}
	return m.queries.ReplaceNIP29ChildOrder(ctx, m.relayScope, groupID, children)
}

func (m *Manager) emitParentStateEvents(ctx context.Context, groupID string) error {
	parentGroupID, exists, err := m.queries.GetNIP29Parent(ctx, m.relayScope, groupID)
	if err != nil || !exists {
		return err
	}
	return m.emitStateEvents(ctx, parentGroupID)
}

func (m *Manager) wouldCreateSubgroupCycle(ctx context.Context, groupID, parentGroupID string) bool {
	for parentGroupID != "" {
		if parentGroupID == groupID {
			return true
		}
		nextParent, exists, err := m.queries.GetNIP29Parent(ctx, m.relayScope, parentGroupID)
		if err != nil || !exists {
			return false
		}
		parentGroupID = nextParent
	}
	return false
}

func (m *Manager) isGroupAdmin(ctx context.Context, groupID, pubkey string) bool {
	roles, err := m.queries.GetNIP29MemberRoleNames(ctx, m.relayScope, groupID, pubkey)
	return err == nil && m.hasAdminRole(roles)
}

func sameGroupIDs(current, requested []string) bool {
	if len(current) != len(requested) {
		return false
	}
	if slices.ContainsFunc(requested, func(groupID string) bool { return groupID == "" }) {
		return false
	}
	seen := make(map[string]struct{}, len(requested))
	for _, groupID := range requested {
		if _, exists := seen[groupID]; exists {
			return false
		}
		seen[groupID] = struct{}{}
	}
	for _, groupID := range current {
		if _, exists := seen[groupID]; !exists {
			return false
		}
	}
	return true
}
