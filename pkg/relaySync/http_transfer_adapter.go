package relaysync

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

func (a *HTTPAdapter) RegisterTransferServices(mux *http.ServeMux, sessions *SessionService, manifests *ManifestService, transfer *TransferService, resolve PrincipalResolver) error {
	if sessions == nil || resolve == nil {
		return ErrUnauthorized
	}
	if manifests != nil {
		mux.HandleFunc("/sync/v2/sessions/{id}/manifest:next", func(w http.ResponseWriter, r *http.Request) { a.handleManifest(w, r, sessions, manifests, resolve) })
	}
	if transfer != nil {
		mux.HandleFunc("/sync/v2/sessions/{id}/events:pull", func(w http.ResponseWriter, r *http.Request) { a.handlePull(w, r, sessions, transfer, resolve) })
		mux.HandleFunc("/sync/v2/sessions/{id}/events:push", func(w http.ResponseWriter, r *http.Request) { a.handlePush(w, r, sessions, transfer, resolve) })
	}
	return nil
}

// RegisterReconcileService registers the bounded Negentropy HTTP endpoint.
func (a *HTTPAdapter) RegisterReconcileService(mux *http.ServeMux, service *ReconcileService, resolve PrincipalResolver) error {
	if service == nil || resolve == nil {
		return ErrUnauthorized
	}
	mux.HandleFunc(sessionsPath+"/{id}/reconcile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		principal, ok := resolve(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := parseSessionPath(r.URL.Path, "/reconcile")
		if err != nil {
			http.Error(w, "invalid session", http.StatusBadRequest)
			return
		}
		var request relaysyncv2.ReconcileRequest
		if err := readProtoBody(r, &request); err != nil {
			http.Error(w, "invalid protobuf", http.StatusBadRequest)
			return
		}
		response, err := service.Reconcile(r.Context(), principal, id, &request)
		if err != nil {
			writeSessionError(w, err)
			return
		}
		writeProto(w, response)
	})
	return nil
}

func parseSessionPath(path, suffix string) (SessionID, error) {
	encoded := strings.TrimSuffix(strings.TrimPrefix(path, sessionsPath+"/"), suffix)
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	var id SessionID
	if err != nil || len(raw) != len(id) {
		return id, ErrInvalidIdentifier
	}
	copy(id[:], raw)
	return id, nil
}
func readProtoBody(r *http.Request, message proto.Message) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(DefaultLimits().MaxBatchBytes)+1))
	if err != nil || len(body) > DefaultLimits().MaxBatchBytes {
		return ErrInvalidLimits
	}
	return proto.Unmarshal(body, message)
}
func (a *HTTPAdapter) handleManifest(w http.ResponseWriter, r *http.Request, sessions *SessionService, service *ManifestService, resolve PrincipalResolver) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	principal, ok := resolve(r)
	if !ok {
		http.Error(w, "unauthorized", 401)
		return
	}
	id, err := parseSessionPath(r.URL.Path, "/manifest:next")
	if err != nil {
		http.Error(w, "invalid session", 400)
		return
	}
	var request relaysyncv2.ManifestRequest
	if err = readProtoBody(r, &request); err != nil {
		http.Error(w, "invalid protobuf", 400)
		return
	}
	response, err := service.Next(r.Context(), principal, id, request.Continuation, request.MaxPartitions)
	if err != nil {
		writeSessionError(w, err)
		return
	}
	writeProto(w, response)
}
func (a *HTTPAdapter) handlePull(w http.ResponseWriter, r *http.Request, sessions *SessionService, service *TransferService, resolve PrincipalResolver) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	principal, ok := resolve(r)
	if !ok {
		http.Error(w, "unauthorized", 401)
		return
	}
	id, err := parseSessionPath(r.URL.Path, "/events:pull")
	if err != nil {
		http.Error(w, "invalid session", 400)
		return
	}
	var request relaysyncv2.PullEventsRequest
	if err = readProtoBody(r, &request); err != nil {
		http.Error(w, "invalid protobuf", 400)
		return
	}
	response, err := service.Pull(r.Context(), principal, id, &request)
	if err != nil {
		writeSessionError(w, err)
		return
	}
	writeProto(w, response)
}
func (a *HTTPAdapter) handlePush(w http.ResponseWriter, r *http.Request, sessions *SessionService, service *TransferService, resolve PrincipalResolver) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	principal, ok := resolve(r)
	if !ok {
		http.Error(w, "unauthorized", 401)
		return
	}
	id, err := parseSessionPath(r.URL.Path, "/events:push")
	if err != nil {
		http.Error(w, "invalid session", 400)
		return
	}
	var request relaysyncv2.PushEventsRequest
	if err = readProtoBody(r, &request); err != nil {
		http.Error(w, "invalid protobuf", 400)
		return
	}
	response, err := service.Push(r.Context(), principal, id, &request)
	if err != nil {
		writeSessionError(w, err)
		return
	}
	writeProto(w, response)
}
