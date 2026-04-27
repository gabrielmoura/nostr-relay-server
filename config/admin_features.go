package config

import (
	"fmt"
	"strings"
)

func (cfg *Config) InternalPort() int {
	if cfg == nil {
		return 0
	}
	return cfg.Port + 1
}

func (cfg *Config) AdminAPIRequiresToken() bool {
	return cfg != nil && strings.TrimSpace(cfg.AdminToken) != ""
}

func (cfg *Config) NIP86Enabled() bool {
	return cfg != nil && cfg.NIP86.Enabled
}

func (cfg *Config) ValidateAdminFeatures() error {
	if cfg == nil || !cfg.NIP86Enabled() {
		return nil
	}
	if strings.TrimSpace(cfg.AdminPubKey) == "" {
		return fmt.Errorf("nip86.enabled requires admin_pubkey")
	}
	if strings.TrimSpace(cfg.RelayInformation.URL) == "" {
		return fmt.Errorf("nip86.enabled requires relay_information.url")
	}
	if cfg.NIP86.AuthWindowSeconds <= 0 {
		return fmt.Errorf("nip86.auth_window_seconds must be greater than zero")
	}
	if cfg.NIP86.CacheTTLSeconds <= 0 {
		return fmt.Errorf("nip86.cache_ttl_seconds must be greater than zero")
	}
	return nil
}
