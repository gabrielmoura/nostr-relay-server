package relaysync

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

const sessionsPath = "/sync/v2/sessions"

type PrincipalResolver func(*http.Request) (Principal, bool)

func (a *HTTPAdapter) RegisterSessionService(mux *http.ServeMux, service *SessionService, resolve PrincipalResolver) error {
	if service == nil || resolve == nil {
		return ErrUnauthorized
	}
	mux.HandleFunc(sessionsPath, func(w http.ResponseWriter, r *http.Request) {
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
		body, err := io.ReadAll(io.LimitReader(r.Body, int64(DefaultLimits().MaxBatchBytes)+1))
		if err != nil || len(body) > DefaultLimits().MaxBatchBytes {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		var request relaysyncv2.OpenSessionRequest
		if err := proto.Unmarshal(body, &request); err != nil {
			http.Error(w, "invalid protobuf", http.StatusBadRequest)
			return
		}
		response, err := service.Open(r.Context(), principal, &request)
		if err != nil {
			writeSessionError(w, err)
			return
		}
		writeProto(w, response)
	})
	mux.HandleFunc(sessionsPath+"/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/complete") {
			a.handleComplete(w, r, service, resolve)
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		principal, ok := resolve(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		encoded := strings.TrimPrefix(r.URL.Path, sessionsPath+"/")
		idBytes, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil || len(idBytes) != len(SessionID{}) {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}
		token := bearerToken(r.Header.Get("Authorization"))
		if len(token) == 0 {
			http.Error(w, "session token required", http.StatusUnauthorized)
			return
		}
		var id SessionID
		copy(id[:], idBytes)
		if err := service.Delete(r.Context(), principal, id, token); err != nil {
			writeSessionError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return nil
}

func (a *HTTPAdapter) handleComplete(w http.ResponseWriter, r *http.Request, service *SessionService, resolve PrincipalResolver) {
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
	id, err := parseSessionPath(r.URL.Path, "/complete")
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if len(token) == 0 {
		http.Error(w, "session token required", http.StatusUnauthorized)
		return
	}
	if r.Body != nil && r.ContentLength != 0 {
		var request relaysyncv2.CompleteSessionRequest
		if err := readProtoBody(r, &request); err != nil || (len(request.SessionId) != 0 && !bytes.Equal(request.SessionId, id[:])) {
			http.Error(w, "invalid protobuf", http.StatusBadRequest)
			return
		}
	}
	if err := service.Complete(r.Context(), principal, id, token); err != nil {
		writeSessionError(w, err)
		return
	}
	writeProto(w, &relaysyncv2.CompleteSessionResponse{Complete: true})
}

func bearerToken(header string) []byte {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
	if err != nil {
		return nil
	}
	return decoded
}
func writeSessionError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if err == ErrUnauthorized || err == ErrSessionToken {
		status = http.StatusUnauthorized
	}
	if err == ErrConflict {
		status = http.StatusConflict
	}
	if err == ErrNotFound {
		status = http.StatusNotFound
	}
	code := relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INTERNAL
	if err == ErrUnauthorized || err == ErrSessionToken {
		code = relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_UNAUTHORIZED
	}
	if err == ErrConflict {
		code = relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INVALID_MESSAGE
	}
	if err == ErrSnapshotExpired {
		code = relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_SNAPSHOT_EXPIRED
	}
	w.Header().Set("Content-Type", protobufMediaType)
	encoded, marshalErr := proto.Marshal(&relaysyncv2.ErrorResponse{Code: code, Message: err.Error(), Retryable: status >= 500})
	if marshalErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func writeProto(w http.ResponseWriter, message proto.Message) {
	encoded, err := proto.Marshal(message)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", protobufMediaType)
	_, _ = w.Write(encoded)
}
