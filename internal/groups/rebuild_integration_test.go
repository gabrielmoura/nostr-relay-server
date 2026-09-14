//go:build integration

package groups

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	storedb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestMigrationsAndNIP29Rebuild(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for integration tests")
	}
	requireIntegrationDatabase(t, dsn)

	ctx := context.Background()
	require.NoError(t, storedb.MigrateDown(ctx, dsn, true))
	require.NoError(t, storedb.MigrateUp(ctx, dsn))
	t.Cleanup(func() {
		require.NoError(t, storedb.MigrateDown(context.Background(), dsn, true))
	})

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	queries := storedb.New(pool)
	groupID := "migration-fixture"
	createGroup := &nostr.Event{
		ID:        strings.Repeat("c", 64),
		PubKey:    strings.Repeat("a", 64),
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindSimpleGroupCreateGroup,
		Tags: nostr.Tags{
			{"d", groupID},
			{"name", "Migration fixture"},
			{"t", "nostr"},
			{"t", "go"},
			{"g", "6gkzwg"},
		},
		Sig: strings.Repeat("b", 128),
	}
	require.NoError(t, queries.InsertEvent(ctx, createGroup))

	manager := &Manager{
		queries: queries,
		cfg: config.NIP29Config{
			GroupCreatorRole: "admin",
			DefaultRoles: []config.NIP29RoleConfig{{
				Name:        "admin",
				Description: "fixture administrator",
				Permissions: []string{"*"},
			}},
			Timeline: config.NIP29TimelineConfig{RecentWindow: 50},
		},
		relayScope:  "wss://migration-test.example",
		roleConfigs: map[string]config.NIP29RoleConfig{},
		roleIDs:     map[string]int32{},
	}
	require.NoError(t, manager.bootstrapRoles(ctx))
	require.NoError(t, manager.rebuildState(ctx))

	group, found, err := queries.GetNIP29Group(ctx, manager.relayScope, groupID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"nostr", "go"}, group.Topics)
	require.Equal(t, []string{"6gkzwg"}, group.Geohashes)
}

func requireIntegrationDatabase(t *testing.T, dsn string) {
	t.Helper()
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	if strings.Contains(strings.ToLower(parsed.Path), "test") {
		return
	}
	t.Fatalf("TEST_DATABASE_URL must target a dedicated test database, got %q", parsed.Path)
}
