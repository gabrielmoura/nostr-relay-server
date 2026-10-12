package relaysync

import (
	"errors"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

const (
	// CapabilitiesPath is the RelaySync v2 capability discovery endpoint.
	CapabilitiesPath = "/sync/v2/capabilities"

	protobufMediaType = "application/x-protobuf"
)

var ErrNilCapabilities = errors.New("relaysync: capabilities are required")

// HTTPAdapter exposes RelaySync's HTTP endpoints without depending on a
// specific server or router. Session endpoints are intentionally not included.
type HTTPAdapter struct {
	capabilities []byte
}

// NewHTTPAdapter creates an HTTP adapter using a snapshot of capabilities.
func NewHTTPAdapter(capabilities *relaysyncv2.Capabilities) (*HTTPAdapter, error) {
	if capabilities == nil {
		return nil, ErrNilCapabilities
	}
	encoded, err := proto.Marshal(capabilities)
	if err != nil {
		return nil, err
	}
	return &HTTPAdapter{capabilities: encoded}, nil
}

// Register adds the RelaySync v2 HTTP endpoints to mux.
func (a *HTTPAdapter) Register(mux *http.ServeMux) {
	mux.HandleFunc(CapabilitiesPath, a.capabilitiesHandler)
}

func (a *HTTPAdapter) capabilitiesHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Vary", "Accept")
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !acceptsProtobuf(request.Header.Values("Accept")) {
		http.Error(writer, "not acceptable", http.StatusNotAcceptable)
		return
	}
	writer.Header().Set("Content-Type", protobufMediaType)
	_, _ = writer.Write(a.capabilities)
}

func acceptsProtobuf(acceptHeaders []string) bool {
	if len(acceptHeaders) == 0 {
		return true
	}
	for _, acceptHeader := range acceptHeaders {
		for _, value := range strings.Split(acceptHeader, ",") {
			value = strings.TrimSpace(value)
			if strings.HasSuffix(value, ";") {
				return false
			}
			mediaType, parameters, err := mime.ParseMediaType(value)
			if err != nil {
				return false
			}
			quality, err := acceptQuality(parameters)
			if err != nil {
				return false
			}
			if quality > 0 && acceptsMediaType(mediaType) {
				return true
			}
		}
	}
	return false
}

func acceptQuality(parameters map[string]string) (float64, error) {
	value, ok := parameters["q"]
	if !ok {
		return 1, nil
	}
	quality, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(quality) || quality < 0 || quality > 1 {
		return 0, errors.New("invalid Accept quality")
	}
	return quality, nil
}

func acceptsMediaType(mediaType string) bool {
	return mediaType == protobufMediaType || mediaType == "application/*" || mediaType == "*/*"
}
