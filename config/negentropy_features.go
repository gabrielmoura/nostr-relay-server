package config

import (
	"fmt"
	"strings"
)

func (cfg *Config) ValidateNegentropyFeatures() error {
	if cfg == nil || !cfg.NegentropyAuth {
		return nil
	}
	if !cfg.EnableNegentropy {
		return fmt.Errorf("negentropy_auth requires enable_negentropy")
	}
	if !cfg.Ws.AuthEnabled() {
		return fmt.Errorf("negentropy_auth requires websocket NIP-42 authentication to be enabled")
	}
	if strings.TrimSpace(cfg.NegentropyAuthorizedPubKey()) == "" {
		return fmt.Errorf("negentropy_auth requires relay_information.pub_key")
	}
	return nil
}

func (cfg *Config) NegentropyAuthorizedPubKey() string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.RelayInformation.PubKey)
}
