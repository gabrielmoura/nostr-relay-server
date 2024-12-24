package policies

import (
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"slices"
)

var publicKinds = []int{
	nostr.KindProfileMetadata,
	nostr.KindFollowList,
	nostr.KindRelayListMetadata,
}

// DownloadOnlyAuthedKinds permite download desses eventos apenas para usuários autenticados.
var DownloadOnlyAuthedKinds = []int{
	nostr.KindApplicationSpecificData,
	nostr.KindEncryptedDirectMessage,
}
var CdnKinds = []int{
	nostr.KindFileStorageServerList,
	nostr.KindUserServerList,
}

func AcceptOnlyPublicKinds(publicKinds []int, except func() bool) func(filters nostr.Filters) bool {
	return func(filters nostr.Filters) bool {
		if except() {
			return true
		}
		for _, filter := range filters {
			for _, kind := range publicKinds {
				if slices.Contains(filter.Kinds, kind) {
					return true
				}
			}
		}

		return false
	}

}

// RejectEventBannedUser rejeita eventos de usuários banidos
func RejectEventBannedUser(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt.PubKey == "" {
		return true, "invalid: missing public key"
	}

	reason, exists, err := db.DbQueries.GetUserBannedByKey(ctx, evt.PubKey)
	if err != nil {
		return true, fmt.Sprintf("error: %s", err.Error())
	}
	if exists {
		return true, fmt.Sprintf("banned: %s", reason)
	}
	return false, ""
}

// RejectReqBannedUser rejeita requisições de usuários banidos, caso a configuração não permita requisições anônimas
func RejectReqBannedUser(ws *dto.WsServer) (bool, string) {
	if !config.Cfg.Relay.EnableAnonymousReq {
		if ws.Authed == "" {
			return true, "invalid: missing public key"
		}

		reason, exists, err := db.DbQueries.GetUserBannedByKey(ws.Ctx, ws.Authed)
		if err != nil {
			return true, fmt.Sprintf("error: %s", err.Error())
		}
		if exists {
			return true, fmt.Sprintf("banned: %s", reason)
		}
	}

	return true, ""
}
