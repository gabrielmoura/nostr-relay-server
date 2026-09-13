package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNIP29TimelineValidationEnabledByDefault(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	setNIP29Defaults()
	if !viper.GetBool("nip29.timeline.enabled") {
		t.Fatal("expected NIP-29 timeline validation to be enabled by default")
	}

	viper.Set("nip29.timeline.enabled", false)
	if viper.GetBool("nip29.timeline.enabled") {
		t.Fatal("expected operators to be able to disable NIP-29 timeline validation")
	}
}
