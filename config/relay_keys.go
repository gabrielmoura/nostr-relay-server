package config

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const nostrHexKeyLength = 64

func (cfg *Config) normalizeRelayKeys() error {
	if cfg == nil {
		return nil
	}

	return cfg.RelayInformation.NormalizeKeys()
}

func (cfg *Config) normalizeAdminKeys() error {
	if cfg == nil {
		return nil
	}

	pubKey, err := normalizeOptionalPublicKey(cfg.AdminPubKey)
	if err != nil {
		return fmt.Errorf("normalize admin_pubkey: %w", err)
	}

	cfg.AdminPubKey = pubKey
	return nil
}

func (cfg *RelayInformationDocument) NormalizeKeys() error {
	if cfg == nil {
		return nil
	}

	pubKey, err := normalizeOptionalPublicKey(cfg.PubKey)
	if err != nil {
		return err
	}

	privKey, err := normalizeOptionalPrivateKey(cfg.PrivKey)
	if err != nil {
		return err
	}

	if privKey != "" {
		derivedPubKey, err := nostr.GetPublicKey(privKey)
		if err != nil {
			return fmt.Errorf("derive relay_information.pub_key from relay_information.priv_key: %w", err)
		}

		if pubKey == "" {
			pubKey = derivedPubKey
		}
	}

	cfg.PubKey = pubKey
	cfg.PrivKey = privKey

	return nil
}

func normalizeOptionalPublicKey(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}

	decoded, err := normalizeKeyValue(value, "npub", "relay_information.pub_key")
	if err != nil {
		return "", err
	}

	return decoded, nil
}

func normalizeOptionalPrivateKey(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}

	decoded, err := normalizeKeyValue(value, "nsec", "relay_information.priv_key")
	if err != nil {
		return "", err
	}

	return decoded, nil
}

func normalizeKeyValue(value string, nip19Prefix string, fieldName string) (string, error) {
	clean := strings.ToLower(strings.TrimSpace(value))
	if clean == "" {
		return "", fmt.Errorf("%s is empty", fieldName)
	}

	if strings.HasPrefix(clean, nip19Prefix) {
		prefix, decoded, err := nip19.Decode(clean)
		if err != nil {
			return "", fmt.Errorf("invalid %s %q: %w", fieldName, clean, err)
		}
		if prefix != nip19Prefix {
			return "", fmt.Errorf("invalid %s prefix %q", fieldName, prefix)
		}

		raw, ok := decoded.(string)
		if !ok || strings.TrimSpace(raw) == "" {
			return "", fmt.Errorf("invalid %s payload", fieldName)
		}

		clean = strings.ToLower(strings.TrimSpace(raw))
	}

	if len(clean) != nostrHexKeyLength {
		return "", fmt.Errorf("invalid %s: expected 64 hex chars, %d received", fieldName, len(clean))
	}

	decoded, err := hex.DecodeString(clean)
	if err != nil {
		return "", fmt.Errorf("invalid %s hex: %w", fieldName, err)
	}
	if len(decoded) != nostrHexKeyLength/2 {
		return "", fmt.Errorf("invalid %s: expected 32 bytes, %d received", fieldName, len(decoded))
	}

	return clean, nil
}
