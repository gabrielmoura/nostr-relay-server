package event

import (
	"strings"

	dbo "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type profileMetadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	About       string `json:"about"`
	Website     string `json:"website"`
	Picture     string `json:"picture"`
	Banner      string `json:"banner"`
	LUD16       string `json:"lud16"`
	NIP05       string `json:"nip05"`
}

func handleProfile(ws *dto.WsServer, evt *nostr.Event) {

	if evt.Kind != nostr.KindProfileMetadata {
		return
	}

	var profile profileMetadata
	if err := json.Unmarshal([]byte(evt.Content), &profile); err != nil {
		log.Logger.Error("failed to parse profile metadata", zap.Error(err))
		return
	}

	err := db.DbQueries.InsertUserProfile(ws.Ctx, &dbo.Profile{
		PublicKey:   evt.PubKey,
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

func checkContainsBot(p profileMetadata) bool {
	return strings.Contains(p.Name, "bot") || strings.Contains(p.DisplayName, "bot") || strings.Contains(p.About, "bot")
}
