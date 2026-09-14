package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNIP70EnabledByDefault(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	setDefaults(false)
	if !viper.GetBool("nip70.enabled") {
		t.Fatal("expected NIP-70 to be enabled by default")
	}
}

func TestDefaultConfigNIP70DisabledDoesNotAdvertiseSupport(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("nip70.enabled", false)

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error = %v", err)
	}
	for _, nip := range cfg.RelayInformation.SupportedNIPs {
		if nip == 70 {
			t.Fatalf("supported NIPs = %v, unexpectedly includes 70", cfg.RelayInformation.SupportedNIPs)
		}
	}
}
