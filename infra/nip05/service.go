package nip05

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

var nip05NamePattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

type Document struct {
	Names  map[string]string   `json:"names"`
	Relays map[string][]string `json:"relays,omitempty"`
}

type Service struct {
	queries *dbmodel.Queries
}

func NewService(queries *dbmodel.Queries) *Service {
	return &Service{queries: queries}
}

func NormalizeName(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return "", fmt.Errorf("missing nip05 name")
	}
	if !nip05NamePattern.MatchString(normalized) {
		return "", fmt.Errorf("invalid nip05 name: only a-z, 0-9, '.', '_' and '-' are allowed")
	}
	return normalized, nil
}

func (s *Service) UpsertIdentity(ctx context.Context, name string, pubkey string) (string, error) {
	if s == nil || s.queries == nil {
		return "", fmt.Errorf("nip05 service is not initialized")
	}

	normalizedName, err := NormalizeName(name)
	if err != nil {
		return "", err
	}

	normalizedPubkey := strings.ToLower(strings.TrimSpace(pubkey))
	if normalizedPubkey == "" {
		return "", fmt.Errorf("missing public key")
	}

	if _, err := s.queries.GetProfileByPublicKey(ctx, normalizedPubkey); err != nil {
		return "", fmt.Errorf("profile not found for pubkey: %w", err)
	}

	if err := s.queries.UpsertNIP05Identity(ctx, normalizedName, normalizedPubkey); err != nil {
		return "", err
	}

	_ = cache.DeleteNIP05Doc()
	return normalizedName, nil
}

func (s *Service) DeleteIdentityByName(ctx context.Context, name string) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("nip05 service is not initialized")
	}

	normalizedName, err := NormalizeName(name)
	if err != nil {
		return err
	}

	if err := s.queries.DeleteNIP05IdentityByName(ctx, normalizedName); err != nil {
		return err
	}

	_ = cache.DeleteNIP05Doc()
	return nil
}

func (s *Service) BuildDocument(ctx context.Context, onlyName string) (*Document, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("nip05 service is not initialized")
	}

	fullDoc, err := s.getOrBuildFullDocument(ctx)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(onlyName) == "" {
		return fullDoc, nil
	}

	normalizedName, err := NormalizeName(onlyName)
	if err != nil {
		return &Document{Names: map[string]string{}}, nil
	}

	pubkey, ok := fullDoc.Names[normalizedName]
	if !ok {
		return &Document{Names: map[string]string{}}, nil
	}

	filtered := &Document{Names: map[string]string{normalizedName: pubkey}}
	if relays, exists := fullDoc.Relays[pubkey]; exists && len(relays) > 0 {
		filtered.Relays = map[string][]string{pubkey: relays}
	}

	return filtered, nil
}

func (s *Service) RelayHintsByPubKeys(ctx context.Context, pubkeys []string) (map[string][]string, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("nip05 service is not initialized")
	}

	normalized := make([]string, 0, len(pubkeys))
	seen := make(map[string]struct{}, len(pubkeys))
	for _, pubkey := range pubkeys {
		pk := strings.ToLower(strings.TrimSpace(pubkey))
		if pk == "" {
			continue
		}
		if _, ok := seen[pk]; ok {
			continue
		}
		seen[pk] = struct{}{}
		normalized = append(normalized, pk)
	}

	eventsByPubkey, err := s.queries.GetLatestRelayListEventsByPubKeys(ctx, normalized)
	if err != nil {
		return nil, err
	}

	relaysByPubkey := make(map[string][]string, len(eventsByPubkey))
	for pubkey, evt := range eventsByPubkey {
		relays := extractRelayHints(evt)
		if len(relays) == 0 {
			continue
		}
		relaysByPubkey[strings.ToLower(pubkey)] = relays
	}

	return relaysByPubkey, nil
}

func (s *Service) getOrBuildFullDocument(ctx context.Context) (*Document, error) {
	if raw, ok := cache.GetNIP05Doc(); ok {
		var cached Document
		if err := json.Unmarshal([]byte(raw), &cached); err == nil {
			if cached.Names == nil {
				cached.Names = map[string]string{}
			}
			return &cached, nil
		}
	}

	identities, err := s.queries.ListNIP05IdentitiesForDocument(ctx)
	if err != nil {
		return nil, err
	}

	doc := &Document{Names: make(map[string]string, len(identities))}
	pubkeys := make([]string, 0, len(identities))
	for _, identity := range identities {
		name := strings.ToLower(strings.TrimSpace(identity.Name))
		pubkey := strings.ToLower(strings.TrimSpace(identity.PublicKey))
		if name == "" || pubkey == "" {
			continue
		}
		doc.Names[name] = pubkey
		pubkeys = append(pubkeys, pubkey)
	}

	relaysByPubkey, err := s.RelayHintsByPubKeys(ctx, pubkeys)
	if err != nil {
		return nil, err
	}
	if len(relaysByPubkey) > 0 {
		doc.Relays = relaysByPubkey
	}

	if payload, err := json.Marshal(doc); err == nil {
		_ = cache.SetNIP05Doc(string(payload))
	}

	return doc, nil
}

func extractRelayHints(event *nostr.Event) []string {
	if event == nil {
		return nil
	}

	hints := make([]string, 0, len(event.Tags))
	seen := make(map[string]struct{}, len(event.Tags))
	for _, tag := range event.Tags {
		if len(tag) < 2 || tag[0] != "r" {
			continue
		}

		relayURL := normalizeRelayHint(tag[1])
		if relayURL == "" {
			continue
		}

		if len(tag) >= 3 {
			marker := strings.ToLower(strings.TrimSpace(tag[2]))
			if marker != "" && marker != "read" && marker != "write" {
				continue
			}
		}

		if _, ok := seen[relayURL]; ok {
			continue
		}
		seen[relayURL] = struct{}{}
		hints = append(hints, relayURL)
	}

	slices.Sort(hints)
	return hints
}

func normalizeRelayHint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "ws" && scheme != "wss" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}

	return parsed.String()
}
