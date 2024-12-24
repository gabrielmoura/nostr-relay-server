package event

import (
	dbo "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk"
	"go.uber.org/zap"
	"strings"
)

func handleProfile(ws *dto.WsServer, evt *nostr.Event) {

	profile, err := sdk.ParseMetadata(evt)
	if err != nil {
		log.Logger.Error("failed to parse profile metadata", zap.Error(err))
		return
	}

	err = db.DbQueries.InsertUserProfile(ws.Ctx, &dbo.Profile{
		PublicKey:   profile.PubKey,
		Name:        profile.Name,
		DisplayName: profile.DisplayName,
		About:       profile.About,
		Website:     profile.Website,
		Picture:     profile.Picture,
		Banner:      profile.Banner,
		Lud16:       profile.LUD16,
		Nip05:       profile.NIP05,
		Bot:         checkContainsBot(profile),
	})
	if err != nil {
		log.Logger.Error("failed to insert profile", zap.Error(err))
	}
}

func checkContainsBot(p sdk.ProfileMetadata) bool {
	return strings.Contains(p.Name, "bot") || strings.Contains(p.DisplayName, "bot") || strings.Contains(p.About, "bot")
}
