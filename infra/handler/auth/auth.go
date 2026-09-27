package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

const AuthContextKey = "authed"

func DoAUTH(ws *dto.WsServer, data dto.Data) string {
	if config.Cfg.Ws.AuthEnabled() {
		var evt nostr.Event
		if err := json.Unmarshal(data[1], &evt); err != nil {
			return "failed to decode auth event: " + err.Error()
		}
		allowedRelayURLs := append([]string{config.Cfg.RelayInformation.CanonicalURL}, privacy.GetActiveAuthURLs()...)
		if pubkey, reason := validateAuthEvent(&evt, ws.Challenge, allowedRelayURLs...); reason == "" {
			ws.Authed = pubkey
			ws.Ctx = context.WithValue(ws.Ctx, AuthContextKey, pubkey)
			ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true}
		} else {
			metrics.NostrRelayAuthFailuresTotal.Inc()
			log.Logger.Warn("failed to authenticate",
				zap.String("reason", reason),
				zap.String("event_id", evt.ID),
				zap.Int("kind", evt.Kind),
				zap.String("pubkey", evt.PubKey),
				zap.String("relay_tag", authRelayTagValue(evt.Tags)),
				zap.String("canonical_url", config.Cfg.RelayInformation.CanonicalURL),
				zap.Time("created_at", evt.CreatedAt.Time()),
				zap.String("remote_ip", ws.RemoteIP),
				zap.String("user_agent", ws.UserAgent),
			)
			ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "error: failed to authenticate"}
		}
	}
	return ""
}

func SendAuthChallenge(ws *dto.WsServer) {
	if !config.Cfg.Ws.AuthEnabled() {
		return
	}
	if ws.Challenge == "" {
		return
	}
	ws.ChanSender <- []any{"AUTH", ws.Challenge}
}

func SendAuthChallengeNow(ws *dto.WsServer) error {
	if !config.Cfg.Ws.AuthEnabled() {
		return nil
	}
	if ws.Challenge == "" {
		return nil
	}
	if err := ws.Conn.WriteJSON([]any{"AUTH", ws.Challenge}); err != nil {
		return fmt.Errorf("failed to send AUTH challenge: %w", err)
	}
	metrics.NostrRelayWsMessagesSend.Inc()
	return nil
}

func validateAuthEvent(evt *nostr.Event, challenge string, relayURLs ...string) (string, string) {
	if evt.Kind != nostr.KindClientAuthentication {
		return "", "invalid_kind"
	}
	if evt.Tags.FindWithValue("challenge", challenge) == nil {
		return "", "challenge_mismatch"
	}

	allowedURLs := make([]*url.URL, 0, len(relayURLs))
	for index, relayURL := range relayURLs {
		expected, parseErr := parseAuthRelayURL(relayURL)
		if parseErr != nil {
			if index == 0 {
				return "", "invalid_canonical_url"
			}
			continue
		}
		allowedURLs = append(allowedURLs, expected)
	}

	relayTag := evt.Tags.Find("relay")
	if relayTag == nil || len(relayTag) < 2 {
		return "", "missing_relay_tag"
	}

	found, err := parseAuthRelayURL(relayTag[1])
	if err != nil {
		return "", "invalid_relay_tag"
	}

	matched := false
	for _, expected := range allowedURLs {
		if authRelayURLsEqual(expected, found) {
			matched = true
			break
		}
	}
	if !matched {
		return "", "relay_mismatch"
	}

	now := time.Now()
	if evt.CreatedAt.Time().After(now.Add(10*time.Minute)) || evt.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {
		return "", "created_at_out_of_window"
	}

	if ok, _ := evt.CheckSignature(); !ok {
		return "", "invalid_signature"
	}

	return evt.PubKey, ""
}

func parseAuthRelayURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSuffix(strings.TrimSpace(raw), "/"))
	if err != nil {
		return nil, err
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	if (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Host == "" {
		return nil, fmt.Errorf("relay URL must be an absolute ws or wss URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("relay URL must not contain credentials, query, or fragment")
	}
	return parsed, nil
}

func authRelayURLsEqual(expected, found *url.URL) bool {
	return expected.Scheme == found.Scheme &&
		strings.EqualFold(expected.Hostname(), found.Hostname()) &&
		authRelayPort(expected) == authRelayPort(found) &&
		expected.Path == found.Path
}

func authRelayPort(relayURL *url.URL) string {
	if port := relayURL.Port(); port != "" {
		return port
	}
	if relayURL.Scheme == "wss" {
		return "443"
	}
	return "80"
}

func authRelayTagValue(tags nostr.Tags) string {
	tag := tags.Find("relay")
	if tag == nil || len(tag) < 2 {
		return ""
	}
	return tag[1]
}
