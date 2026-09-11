package http

import (
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

func TestNIP11WithPrivacy_AdvertisesNIP29(t *testing.T) {
	t.Parallel()

	doc, ok := NIP11WithPrivacy(&config.Config{
		NIP29: config.NIP29Config{Enabled: true},
		RelayInformation: config.RelayInformationDocument{
			SupportedNIPs: []int{1, 11},
		},
	}).(map[string]any)
	if !ok {
		t.Fatal("expected augmented NIP-11 document")
	}
	if _, ok := doc["nip29"].(map[string]any); !ok {
		t.Fatal("expected nip29 support object")
	}

	supported, ok := doc["supported_nips"].([]any)
	if !ok {
		t.Fatalf("supported_nips type = %T", doc["supported_nips"])
	}
	for _, nip := range supported {
		if isNIP29Number(nip) {
			return
		}
	}
	t.Fatalf("supported_nips = %v, want 29", supported)
}
