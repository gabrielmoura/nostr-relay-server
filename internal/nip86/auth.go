package nip86

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

const authScheme = "Nostr "

func Authenticate(cfg *config.Config, input AuthInput) (AuthResult, error) {
	if cfg == nil || !cfg.NIP86Enabled() {
		return AuthResult{}, errors.New("nip86 is disabled")
	}
	if cfg.AdminPubKey == "" {
		return AuthResult{}, errors.New("admin_pubkey is not configured")
	}
	if !strings.HasPrefix(input.Authorization, authScheme) {
		return AuthResult{}, errors.New("missing or invalid Authorization header")
	}

	token := strings.TrimSpace(strings.TrimPrefix(input.Authorization, authScheme))
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return AuthResult{}, fmt.Errorf("decode Authorization header: %w", err)
	}

	var event nostr.Event
	if err := jsonx.Unmarshal(decoded, &event); err != nil {
		return AuthResult{}, fmt.Errorf("decode authorization event: %w", err)
	}
	if event.Kind != 27235 {
		return AuthResult{}, errors.New("authorization event kind must be 27235")
	}
	if ok, err := event.CheckSignature(); err != nil || !ok {
		return AuthResult{}, errors.New("authorization event signature is invalid")
	}
	if err := validateFreshness(cfg, event.CreatedAt.Time()); err != nil {
		return AuthResult{}, err
	}
	if !strings.EqualFold(tagValue(event.Tags, "method"), input.Method) {
		return AuthResult{}, errors.New("authorization method tag mismatch")
	}
	if !sameAbsoluteURL(tagValue(event.Tags, "u"), input.URL) {
		return AuthResult{}, errors.New("authorization u tag mismatch")
	}
	if err := validatePayloadTag(event.Tags, input.Body); err != nil {
		return AuthResult{}, err
	}
	if !strings.EqualFold(event.PubKey, cfg.AdminPubKey) {
		return AuthResult{}, errors.New("caller is not the configured relay administrator")
	}

	return AuthResult{PubKey: strings.ToLower(event.PubKey), Event: event}, nil
}

func validateFreshness(cfg *config.Config, createdAt time.Time) error {
	window := time.Duration(cfg.NIP86.AuthWindowSeconds) * time.Second
	if window <= 0 {
		window = time.Minute
	}
	now := time.Now().UTC()
	if createdAt.Before(now.Add(-window)) || createdAt.After(now.Add(window)) {
		return errors.New("authorization event is outside the allowed time window")
	}
	return nil
}

func validatePayloadTag(tags nostr.Tags, body []byte) error {
	payload := tagValue(tags, "payload")
	if payload == "" {
		return errors.New("authorization payload tag is required")
	}
	sum := sha256.Sum256(body)
	if payload != hex.EncodeToString(sum[:]) {
		return errors.New("authorization payload hash mismatch")
	}
	return nil
}

func tagValue(tags nostr.Tags, key string) string {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == key {
			return strings.TrimSpace(tag[1])
		}
	}
	return ""
}

func sameAbsoluteURL(expected, actual string) bool {
	eu, err := url.Parse(strings.TrimSpace(expected))
	if err != nil {
		return false
	}
	au, err := url.Parse(strings.TrimSpace(actual))
	if err != nil {
		return false
	}

	if !strings.EqualFold(eu.Scheme, au.Scheme) || !strings.EqualFold(eu.Host, au.Host) {
		return false
	}
	if normalizeURLPath(eu.Path) != normalizeURLPath(au.Path) {
		return false
	}
	if eu.RawQuery != au.RawQuery {
		return false
	}
	return bytes.Equal([]byte(eu.Fragment), []byte(au.Fragment))
}

func normalizeURLPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}
	return strings.TrimRight(trimmed, "/")
}
