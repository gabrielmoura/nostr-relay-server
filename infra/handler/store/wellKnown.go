package store

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"net/http"
)

func HandleWellKnown(w http.ResponseWriter, r *http.Request) {
	// check if the request is for the well-known file
	if r.URL.Path == "/.well-known/nostr.json" {
		net.JsonResponse(w, http.StatusOK, map[string]any{
			"media": map[string]any{
				"apiPath":           config.Cfg.Store.APIPath,
				"mediaPath":         config.Cfg.Store.MediaPath,
				"acceptedMimetypes": config.Cfg.Store.AcceptedMimetypes,
				"contentPolicy": map[string]any{
					"allowAdultContent":   config.Cfg.Store.AllowAdultContent,
					"allowViolentContent": config.Cfg.Store.AllowViolentContent,
				},
			},
			"names": config.Cfg.Store.Names,
		})
	}

	http.Error(w, "Not Found", http.StatusNotFound)

}

func HandleWellKnownNip96(w http.ResponseWriter, r *http.Request) {
	net.JsonResponse(w, http.StatusOK, config.FileServerConfig{
		APIURL:      config.Cfg.Store.APIPath,
		DownloadURL: config.Cfg.Store.MediaPath,
	})
}
