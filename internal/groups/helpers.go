package groups

import (
	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

func buildMetadataTags(group *dbstore.NIP29Group) nostr.Tags {
	tags := nostr.Tags{{"d", group.GroupID}}
	appendOptionalTag(&tags, "name", group.Name)
	appendOptionalTag(&tags, "picture", group.Picture)
	appendOptionalTag(&tags, "about", group.About)
	appendStatusTag(&tags, "private", "public", group.Private)
	appendStatusTag(&tags, "closed", "open", group.Closed)
	appendMarkerTag(&tags, "restricted", group.Restricted)
	appendMarkerTag(&tags, "hidden", group.Hidden)
	return tags
}

func appendOptionalTag(tags *nostr.Tags, key, value string) {
	if value != "" {
		*tags = append(*tags, nostr.Tag{key, value})
	}
}

func appendMarkerTag(tags *nostr.Tags, key string, enabled bool) {
	if enabled {
		*tags = append(*tags, nostr.Tag{key})
	}
}

func appendStatusTag(tags *nostr.Tags, trueTag, falseTag string, enabled bool) {
	if enabled {
		*tags = append(*tags, nostr.Tag{trueTag})
	} else {
		*tags = append(*tags, nostr.Tag{falseTag})
	}
}

func groupIDFromEvent(evt *nostr.Event) string {
	if evt == nil {
		return ""
	}
	if value := firstTagValue(evt, "h"); value != "" {
		return value
	}
	return firstTagValue(evt, "d")
}

func firstTagValue(evt *nostr.Event, key string) string {
	for _, tag := range evt.Tags.GetAll([]string{key, ""}) {
		if len(tag) > 1 {
			return tag[1]
		}
	}
	return ""
}

func allTagValues(evt *nostr.Event, key string) []string {
	values := make([]string, 0, 4)
	for _, tag := range evt.Tags.GetAll([]string{key}) {
		if len(tag) > 1 {
			values = append(values, tag[1:]...)
		}
	}
	return values
}

func tagExists(evt *nostr.Event, key string) bool {
	return evt.Tags.GetFirst([]string{key}) != nil
}

func isValidGroupID(groupID string) bool {
	if groupID == "" {
		return false
	}
	for _, r := range groupID {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func memberStateTags(groupID string, memberRoles []dbstore.NIP29MemberRole, isAdmin func([]string) bool) (nostr.Tags, nostr.Tags) {
	byUser := make(map[string][]string)
	for _, item := range memberRoles {
		byUser[item.UserID] = append(byUser[item.UserID], item.RoleName)
	}

	adminsTags := nostr.Tags{{"d", groupID}}
	membersTags := nostr.Tags{{"d", groupID}}
	for userID, roles := range byUser {
		membersTags = append(membersTags, nostr.Tag{"p", userID})
		if !isAdmin(roles) {
			continue
		}
		tag := nostr.Tag{"p", userID}
		tag = append(tag, roles...)
		adminsTags = append(adminsTags, tag)
	}
	return adminsTags, membersTags
}

func roleStateTags(groupID string, roles []dbstore.NIP29Role) nostr.Tags {
	tags := nostr.Tags{{"d", groupID}}
	for _, role := range roles {
		tag := nostr.Tag{"role", role.Name}
		if role.Description != "" {
			tag = append(tag, role.Description)
		}
		tags = append(tags, tag)
	}
	return tags
}
