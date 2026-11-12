package event

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestParseDiscoveryRequest(t *testing.T) {
	pubkey := strings.Repeat("a", 64)
	req, err := parseDiscoveryRequest(&nostr.Event{PubKey: pubkey, Tags: nostr.Tags{{"param", "max_results", "7"}}})
	require.NoError(t, err)
	require.Equal(t, pubkey, req.pubKey)
	require.Equal(t, 7, req.limit)

	_, err = parseDiscoveryRequest(&nostr.Event{PubKey: pubkey, Tags: nostr.Tags{{"param", "max_results", "0"}}})
	require.Error(t, err)
	_, err = parseDiscoveryRequest(&nostr.Event{PubKey: pubkey, Tags: nostr.Tags{{"param", "p", pubkey}, {"param", "p", strings.Repeat("b", 64)}}})
	require.Error(t, err)
	_, err = parseDiscoveryRequest(&nostr.Event{PubKey: pubkey, Tags: nostr.Tags{{"param", "p", pubkey}, {"param", "user", strings.Repeat("b", 64)}}})
	require.Error(t, err)
}

func TestHasDiscoveryEncryption(t *testing.T) {
	require.True(t, hasDiscoveryEncryption(nostr.Tags{{"encrypted"}}))
	require.True(t, hasDiscoveryEncryption(nostr.Tags{{"param", "encrypted", "true"}}))
	require.False(t, hasDiscoveryEncryption(nostr.Tags{{"param", "max_results", "10"}}))
}

func TestDiscoveryReservationBoundsConcurrencyAndReleases(t *testing.T) {
	requests := []*nostr.Event{{ID: "one"}, {ID: "two"}, {ID: "three"}, {ID: "four"}}
	for _, request := range requests {
		require.True(t, reserveDiscovery(request))
	}
	require.False(t, reserveDiscovery(requests[0]))
	require.False(t, reserveDiscovery(&nostr.Event{ID: "five"}))
	for _, request := range requests {
		releaseDiscovery(request.ID)
	}
	require.True(t, reserveDiscovery(requests[0]))
	releaseDiscovery(requests[0].ID)
}

func TestContentDiscoveryResponseUsesNIP90ContentAndReferences(t *testing.T) {
	if config.Cfg == nil {
		config.Cfg = &config.Config{}
	}
	previous := config.Cfg.RelayInformation
	defer func() { config.Cfg.RelayInformation = previous }()
	privateKey := nostr.GeneratePrivateKey()
	pubkey, err := nostr.GetPublicKey(privateKey)
	require.NoError(t, err)
	config.Cfg.RelayInformation.PrivKey = privateKey
	config.Cfg.RelayInformation.PubKey = pubkey
	request := &nostr.Event{ID: strings.Repeat("1", 64), PubKey: strings.Repeat("2", 64), Tags: nostr.Tags{{"i", "input", "text"}}}
	response, err := contentDiscoveryResponse(request, []string{strings.Repeat("3", 64)})
	require.NoError(t, err)
	valid, err := response.CheckSignature()
	require.NoError(t, err)
	require.True(t, valid)
	require.Contains(t, response.Tags, nostr.Tag{"e", request.ID})
	require.Contains(t, response.Tags, nostr.Tag{"p", request.PubKey})
	require.Contains(t, response.Tags, nostr.Tag{"i", "input", "text"})
	var content nostr.Tags
	require.NoError(t, json.Unmarshal([]byte(response.Content), &content))
	require.Equal(t, nostr.Tags{{"e", strings.Repeat("3", 64)}}, content)
}

func TestFollowedPubkeysDeduplicatesAndRejectsInvalidValues(t *testing.T) {
	pubkey := strings.Repeat("a", 64)
	require.Equal(t, []string{pubkey}, followedPubkeys(nostr.Tags{{"p", pubkey}, {"p", pubkey}, {"p", "invalid"}}))
}
