package db

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpsertNIP29GroupParameterMapping(t *testing.T) {
	deletedAt := time.Date(2026, time.September, 14, 2, 0, 0, 0, time.UTC)
	lastMetadataUpdate := time.Date(2026, time.September, 14, 2, 1, 0, 0, time.UTC)
	lastAdminsUpdate := time.Date(2026, time.September, 14, 2, 2, 0, 0, time.UTC)
	lastMembersUpdate := time.Date(2026, time.September, 14, 2, 3, 0, 0, time.UTC)
	lastRolesUpdate := time.Date(2026, time.September, 14, 2, 4, 0, 0, time.UTC)
	group := NIP29Group{
		Relay:                        "wss://relay.example",
		GroupID:                      "group",
		Name:                         "Group",
		Picture:                      "picture",
		About:                        "about",
		Topics:                       []string{"nostr"},
		Geohashes:                    []string{"6gkzwg"},
		Private:                      true,
		Closed:                       true,
		Restricted:                   true,
		Hidden:                       true,
		CreatedBy:                    strings.Repeat("a", 64),
		DeletedAt:                    &deletedAt,
		MinPoW:                       20,
		RequireModerationTimelineRef: true,
		MinTimelineReferences:        2,
		TimelineRecentWindow:         50,
		AllowLatePublication:         true,
		LastMetadataUpdate:           lastMetadataUpdate,
		LastAdminsUpdate:             lastAdminsUpdate,
		LastMembersUpdate:            lastMembersUpdate,
		LastRolesUpdate:              lastRolesUpdate,
	}

	args := upsertNIP29GroupArgs(group)
	require.Len(t, args, 22)
	require.Equal(t, group.CreatedBy, args[11])
	require.Equal(t, group.DeletedAt, args[12])
	require.Equal(t, group.LastMetadataUpdate, args[18])
	require.Equal(t, group.LastRolesUpdate, args[21])
	require.Contains(t, upsertNIP29Group, "$11,$12,NOW(),$13,$14,$15,$16,$17,$18,$19,$20,$21,$22")
}

func TestUpsertNIP29GroupParameterMappingUsesEmptyArrays(t *testing.T) {
	args := upsertNIP29GroupArgs(NIP29Group{})

	topics, ok := args[5].([]string)
	require.True(t, ok)
	require.Empty(t, topics)
	require.NotNil(t, topics)

	geohashes, ok := args[6].([]string)
	require.True(t, ok)
	require.Empty(t, geohashes)
	require.NotNil(t, geohashes)
}
