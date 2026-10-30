// Package nethttp adapts RelaySync to the standard net/http router.
// It does not create a listener and leaves TLS/HTTP versions to the caller.
package nethttp

import (
	"net/http"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

type Services struct {
	Sessions  *relaysync.SessionService
	Manifests *relaysync.ManifestService
	Transfer  *relaysync.TransferService
	Reconcile *relaysync.ReconcileService
	Resolve   relaysync.PrincipalResolver
}

// NewHandler builds a standard-library handler for the configured services.
// The returned handler may be mounted below an existing router or server.
func NewHandler(adapter *relaysync.HTTPAdapter, services Services) (http.Handler, error) {
	if adapter == nil || services.Resolve == nil {
		return nil, relaysync.ErrUnauthorized
	}
	mux := http.NewServeMux()
	adapter.Register(mux)
	if services.Sessions != nil {
		if err := adapter.RegisterSessionService(mux, services.Sessions, services.Resolve); err != nil {
			return nil, err
		}
	}
	if services.Manifests != nil || services.Transfer != nil {
		if err := adapter.RegisterTransferServices(mux, services.Sessions, services.Manifests, services.Transfer, services.Resolve); err != nil {
			return nil, err
		}
	}
	if services.Reconcile != nil {
		if err := adapter.RegisterReconcileService(mux, services.Reconcile, services.Resolve); err != nil {
			return nil, err
		}
	}
	return mux, nil
}
