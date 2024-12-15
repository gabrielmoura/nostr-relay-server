package handler

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/goccy/go-json"
	"net/http"
)

func handleRelayInfo(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(config.Cfg.RelayInformation)
	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}
