package policies

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/nbd-wtf/go-nostr"
)

const (
	marmotKindKeyPackage     = 30443
	marmotKindLegacyPackage  = 443
	marmotKindKeyPackageList = 10051

	marmotExtensionGroupData  = "0xf2ee"
	marmotExtensionLastResort = "0x000a"
	marmotProposalSelfRemove  = "0x000a"
)

func (p Policies) validateMarmotMIP00Event(evt *nostr.Event) (bool, string) {
	if p.Config == nil || !p.Config.Marmot.Enabled || !p.Config.Marmot.MIP00.Enabled {
		return false, ""
	}

	mode := p.Config.Marmot.MIP00.NormalizedValidationMode()
	if mode == "off" {
		return false, ""
	}
	kindLabel := strconv.Itoa(evt.Kind)

	switch evt.Kind {
	case marmotKindKeyPackage:
		if !p.Config.Marmot.MIP00.AcceptKind30443 {
			metrics.NostrMarmotMIP00EventsTotal.WithLabelValues(kindLabel, "rejected").Inc()
			return true, security.Reason(security.PrefixRestricted, "marmot mip00 keypackage events are disabled")
		}
		reject, reason := validateMarmotKeyPackage(evt, p.Config.Marmot.MIP00)
		return validateAndRecordMarmotMIP00(kindLabel, reject, reason)
	case marmotKindKeyPackageList:
		if !p.Config.Marmot.MIP00.AcceptKind10051 {
			metrics.NostrMarmotMIP00EventsTotal.WithLabelValues(kindLabel, "rejected").Inc()
			return true, security.Reason(security.PrefixRestricted, "marmot mip00 relay list events are disabled")
		}
		reject, reason := validateMarmotRelayList(evt, p.Config.Marmot.MIP00)
		return validateAndRecordMarmotMIP00(kindLabel, reject, reason)
	case marmotKindLegacyPackage:
		if !p.Config.Marmot.MIP00.AcceptLegacyKind443 {
			return false, ""
		}
		reject, reason := validateMarmotLegacyKeyPackage(evt, p.Config.Marmot.MIP00)
		return validateAndRecordMarmotMIP00(kindLabel, reject, reason)
	default:
		return false, ""
	}
}

func validateAndRecordMarmotMIP00(kind string, reject bool, reason string) (bool, string) {
	recordMarmotMIP00Result(kind, reject)
	return reject, reason
}

func recordMarmotMIP00Result(kind string, reject bool) {
	result := "accepted"
	if reject {
		result = "rejected"
	}
	metrics.NostrMarmotMIP00EventsTotal.WithLabelValues(kind, result).Inc()
}

func validateMarmotKeyPackage(evt *nostr.Event, cfg config.MarmotMIP00Config) (bool, string) {
	return validateMarmotBaseKeyPackage(evt, cfg, false)
}

func validateMarmotLegacyKeyPackage(evt *nostr.Event, cfg config.MarmotMIP00Config) (bool, string) {
	legacyCfg := cfg
	legacyCfg.RequireITag = false
	return validateMarmotBaseKeyPackage(evt, legacyCfg, true)
}

func validateMarmotBaseKeyPackage(evt *nostr.Event, cfg config.MarmotMIP00Config, legacy bool) (bool, string) {
	if !legacy {
		if value, ok := firstTagValue(evt.Tags, "d"); !ok || strings.TrimSpace(value) == "" {
			return true, marmotInvalid("keypackage missing d tag")
		}
	}
	if cfg.RequireBase64EncodingTag {
		encoding, ok := firstTagValue(evt.Tags, "encoding")
		if !ok {
			return true, marmotInvalid("keypackage missing encoding tag")
		}
		if strings.TrimSpace(encoding) != "base64" {
			return true, marmotInvalid("keypackage encoding must be base64")
		}
	}
	if _, ok := firstTagValue(evt.Tags, "mls_protocol_version"); !ok {
		return true, marmotInvalid("keypackage missing mls_protocol_version tag")
	}
	if _, ok := firstTagValue(evt.Tags, "mls_ciphersuite"); !ok {
		return true, marmotInvalid("keypackage missing mls_ciphersuite tag")
	}
	if len(evt.Content) > cfg.MaxContentSizeBytes {
		return true, marmotInvalid("keypackage content exceeds configured size")
	}
	if cfg.RequireITag {
		value, ok := firstTagValue(evt.Tags, "i")
		if !ok || strings.TrimSpace(value) == "" {
			return true, marmotInvalid("keypackage missing required i tag")
		}
		if !isHex(value) {
			return true, marmotInvalid("keypackage i tag must be valid hex")
		}
	}
	if cfg.RequireRelaysTag {
		relayValues, ok := tagValues(evt.Tags, "relays")
		if !ok || len(relayValues) == 0 {
			return true, marmotInvalid("keypackage missing relays tag")
		}
		if reject, reason := validateRelayValues("keypackage", relayValues, cfg); reject {
			return true, reason
		}
	}
	if cfg.RequireMLSExtensions {
		values, ok := tagValues(evt.Tags, "mls_extensions")
		if !ok || len(values) == 0 {
			return true, marmotInvalid("keypackage missing mls_extensions tag")
		}
		if !containsValue(values, marmotExtensionGroupData) || !containsValue(values, marmotExtensionLastResort) {
			return true, marmotInvalid("keypackage mls_extensions missing required marmot extension ids")
		}
	}
	if cfg.RequireMLSProposals {
		values, ok := tagValues(evt.Tags, "mls_proposals")
		if !ok || len(values) == 0 {
			return true, marmotInvalid("keypackage missing mls_proposals tag")
		}
		if !containsValue(values, marmotProposalSelfRemove) {
			return true, marmotInvalid("keypackage mls_proposals missing required self_remove id")
		}
	}

	return false, ""
}

func validateMarmotRelayList(evt *nostr.Event, cfg config.MarmotMIP00Config) (bool, string) {
	relayTags := make([]string, 0, len(evt.Tags))
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "relay" {
			relayTags = append(relayTags, tag[1])
		}
	}
	if len(relayTags) == 0 {
		return true, marmotInvalid("relay list missing relay tags")
	}
	return validateRelayValues("relay list", relayTags, cfg)
}

func validateRelayValues(eventType string, values []string, cfg config.MarmotMIP00Config) (bool, string) {
	if len(values) > cfg.MaxRelaysPerEvent {
		return true, marmotInvalid(eventType + " exceeds configured relay limit")
	}
	if !cfg.RequireWSRelayURLs {
		return false, ""
	}
	for _, value := range values {
		url := strings.ToLower(strings.TrimSpace(value))
		if strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://") {
			continue
		}
		return true, marmotInvalid(eventType + " contains invalid relay url")
	}
	return false, ""
}

func marmotInvalid(message string) string {
	return security.Reason(security.PrefixInvalid, "marmot mip00 "+message)
}

func firstTagValue(tags nostr.Tags, name string) (string, bool) {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == name {
			return tag[1], true
		}
	}
	return "", false
}

func tagValues(tags nostr.Tags, name string) ([]string, bool) {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == name {
			values := make([]string, 0, len(tag)-1)
			for _, value := range tag[1:] {
				trimmed := strings.TrimSpace(value)
				if trimmed != "" {
					values = append(values, trimmed)
				}
			}
			return values, true
		}
	}
	return nil, false
}

func containsValue(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func isHex(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value)%2 != 0 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
