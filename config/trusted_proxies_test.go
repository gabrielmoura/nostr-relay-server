package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestValidateTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		proxies []string
		wantErr bool
	}{
		{name: "empty list"},
		{name: "ip and cidr", proxies: []string{"127.0.0.1", "::1", "10.0.0.0/8", "2001:db8::/32"}},
		{name: "empty entry", proxies: []string{""}, wantErr: true},
		{name: "whitespace entry", proxies: []string{" 10.0.0.1"}, wantErr: true},
		{name: "invalid entry", proxies: []string{"proxy.internal"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Config{Security: SecurityConfig{TrustedProxies: tt.proxies}}).ValidateTrustedProxies()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTrustedProxies() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestApplyLoadedConfigValidatesTrustedProxies(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	previousCfg := Cfg
	t.Cleanup(func() { Cfg = previousCfg })

	setDefaults(false)
	viper.Set("db.postgres_uri", "postgres://example.invalid/nostr")
	viper.Set("security.trusted_proxies", []string{" proxy.internal"})

	err := applyLoadedConfig()
	if err == nil || !strings.Contains(err.Error(), "security.trusted_proxies") {
		t.Fatalf("applyLoadedConfig() error = %v, want trusted proxy validation error", err)
	}
}
