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
	appendMarkerTag(&tags, "restricted", group.Restricted)
	appendMarkerTag(&tags, "hidden", group.Hidden)
	appendMarkerTag(&tags, "private", group.Private)
	appendMarkerTag(&tags, "closed", group.Closed)
	for _, topic := range group.Topics {
		appendOptionalTag(&tags, "t", topic)
	}
	for _, geohash := range group.Geohashes {
		appendOptionalTag(&tags, "g", geohash)
	}
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

// visibilityFlags maps Amethyst's visibility value to NIP-29 metadata flags.
// Public/open access is represented by the absence of every marker.
func visibilityFlags(value string) (private, closed, restricted, hidden bool, ok bool) {
	switch value {
	case "public", "open":
		return false, false, false, false, true
	case "private":
		return true, false, false, false, true
	case "closed":
		return false, true, false, false, true
	case "restricted":
		return false, false, true, false, true
	case "hidden":
		return false, false, false, true, true
	default:
		return false, false, false, false, false
	}
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
