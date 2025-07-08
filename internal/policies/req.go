package policies

import (
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
	"slices"
)

type Policies struct {
	Config *config.Config
}

var P *Policies

func Init() {
	P = &Policies{
		Config: config.Cfg,
	}
}

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

func (p Policies) AcceptOnlyPublicKinds(publicKinds []int, except func() bool) func(filters nostr.Filters) bool {
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
func (p Policies) RejectEventBannedUser(ctx context.Context, evt *nostr.Event) (bool, string) {
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
func (p Policies) RejectReqBannedUser(ws *dto.WsServer) (bool, string) {
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

// NoEmptyFilters disallows filters that don't have at least a tag, a kind, an author or an id.
func (p Policies) NoEmptyFilters(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
	c := len(filter.Kinds) + len(filter.IDs) + len(filter.Authors)
	for _, tagItems := range filter.Tags {
		c += len(tagItems)
	}
	if c == 0 {
		return true, "can't handle empty filters"
	}
	return false, ""
}

// AntiSyncBots tries to prevent people from syncing kind:1s from this relay to else by always
// requiring an author parameter at least.
func (p Policies) AntiSyncBots(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
	if p.Config.Ws.Auth {
		return (len(filter.Kinds) == 0 || slices.Contains(filter.Kinds, 1)) &&
			len(filter.Authors) == 0, "auth-required: an author must be specified to get their kind:1 notes"
	}
	return false, ""
}

// CheckKindsAuth Verifica se
func (p Policies) CheckKindsAuth(filter nostr.Filter, ws *dto.WsServer) (reject bool, msg string) {
	receivers, _ := filter.Tags["p"]
	if !slices.Contains(filter.Authors, ws.Authed) || !slices.Contains(receivers, ws.Authed) {
		return false, ""
	}
	for kind := range p.Config.Relay.ProtectedKinds {
		if slices.Contains(filter.Kinds, kind) {
			return true,
				fmt.Sprintf("auth-required: é necessário autenticação para acessar eventos do tipo %s", metrics.GetKindName(kind))
		}
	}
	return false, ""
}
func (p Policies) AcceptReqs(filters nostr.Filters, ws *dto.WsServer) bool {
	if config.Cfg.Ws.Auth {
		if ws.Authed != "" {
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
	return true
}
func (p Policies) CheckMinimumPow(evt nostr.Event) (reject bool, msg string) {
	if p.Config.Relay.MinimumPOWLimit == 0 {
		return false, ""
	}
	err := nip13.Check(evt.ID, config.Cfg.Relay.MinimumPOWLimit)
	if err != nil {
		return true, "blocked: minimum POW not obtained"
	} else {
		return false, ""
	}
}
