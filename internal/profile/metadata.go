// Package profile materializes Nostr profile metadata after an event is stored.
package profile

import (
	"context"
	"strings"

	dbo "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

type metadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	About       string `json:"about"`
	Website     string `json:"website"`
	Picture     string `json:"picture"`
	Banner      string `json:"banner"`
	LUD16       string `json:"lud16"`
	NIP05       string `json:"nip05"`
}

// Upsert stores the materialized view for a persisted kind 0 event.
func Upsert(ctx context.Context, evt *nostr.Event) error {
	if evt == nil || evt.Kind != nostr.KindProfileMetadata {
		return nil
	}

	var parsed metadata
	if err := json.Unmarshal([]byte(evt.Content), &parsed); err != nil {
		return err
	}

	return db.DbQueries.InsertUserProfile(ctx, &dbo.Profile{
		PublicKey:   evt.PubKey,
		Name:        parsed.Name,
		DisplayName: parsed.DisplayName,
		About:       parsed.About,
		Website:     parsed.Website,
		Picture:     parsed.Picture,
		Banner:      parsed.Banner,
		Lud16:       parsed.LUD16,
		Nip05:       parsed.NIP05,
		Bot:         containsBot(parsed),
	})
}

func containsBot(parsed metadata) bool {
	return strings.Contains(parsed.Name, "bot") ||
		strings.Contains(parsed.DisplayName, "bot") ||
		strings.Contains(parsed.About, "bot")
}
