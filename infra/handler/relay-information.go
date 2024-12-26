package handler

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"net/http"
)

func handleRelayInfo(w http.ResponseWriter, r *http.Request) {
	net.JsonResponse(w, http.StatusOK, config.Cfg.RelayInformation)
	return
}
