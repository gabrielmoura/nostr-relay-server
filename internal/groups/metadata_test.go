package groups

import (
	"testing"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

func TestApplyMetadataEdits_Visibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility string
		private    bool
		closed     bool
		restricted bool
		hidden     bool
	}{
		{name: "public", visibility: "public"},
		{name: "private", visibility: "private", private: true},
		{name: "closed", visibility: "closed", closed: true},
		{name: "restricted", visibility: "restricted", restricted: true},
		{name: "hidden", visibility: "hidden", hidden: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			group := &dbstore.NIP29Group{Private: true, Closed: true, Restricted: true, Hidden: true}
			evt := &nostr.Event{Tags: nostr.Tags{{"visibility", tt.visibility}}}
			applyMetadataEdits(group, evt)

			if group.Private != tt.private || group.Closed != tt.closed || group.Restricted != tt.restricted || group.Hidden != tt.hidden {
				t.Fatalf("visibility %q produced private=%t closed=%t restricted=%t hidden=%t", tt.visibility, group.Private, group.Closed, group.Restricted, group.Hidden)
			}
		})
	}
}

func TestBuildMetadataTags_UsesOnlyNIP29VisibilityMarkers(t *testing.T) {
	t.Parallel()

	tags := buildMetadataTags(&dbstore.NIP29Group{GroupID: "group", Private: true, Restricted: true})
	for _, tag := range tags {
		if tag[0] == "public" || tag[0] == "open" || tag[0] == "visibility" {
			t.Fatalf("unexpected non-NIP-29 visibility tag %q", tag[0])
		}
	}

	if !tagExists(&nostr.Event{Tags: tags}, "private") || !tagExists(&nostr.Event{Tags: tags}, "restricted") {
		t.Fatal("expected enabled NIP-29 visibility markers")
	}
}

func TestApplyMetadataEdits_PreservesTopicsAndGeohashes(t *testing.T) {
	t.Parallel()

	group := &dbstore.NIP29Group{}
	evt := &nostr.Event{Tags: nostr.Tags{{"t", "nostr"}, {"t", "go"}, {"g", "6gkzwg"}}}
	applyMetadataEdits(group, evt)

	metadata := &nostr.Event{Tags: buildMetadataTags(group)}
	if got := allTagValues(metadata, "t"); len(got) != 2 || got[0] != "nostr" || got[1] != "go" {
		t.Fatalf("topics = %v", got)
	}
	if got := allTagValues(metadata, "g"); len(got) != 1 || got[0] != "6gkzwg" {
		t.Fatalf("geohashes = %v", got)
	}
}

func TestResolveRoleIDs_DoesNotAddMemberToPrivilegedRole(t *testing.T) {
	t.Parallel()

	m := &Manager{memberRoleID: 1, roleIDs: map[string]int32{"admin": 2}}
	roles := m.resolveRoleIDs([]string{"admin"})
	if len(roles) != 1 || roles[0] != 2 {
		t.Fatalf("roles = %v, want only admin role", roles)
	}
}

func TestPinnedEventKindIsRelayGeneratedMetadata(t *testing.T) {
	t.Parallel()

	if !isNIP29MetadataKind(kindSimpleGroupPinnedEvents) {
		t.Fatal("expected kind 39005 to be treated as relay-generated metadata")
	}
}
