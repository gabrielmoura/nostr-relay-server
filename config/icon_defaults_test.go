package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultsLeaveRelayIconUnset(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	setDefaults(false)
	if got := viper.GetString("relay_information.icon"); got != "" {
		t.Fatalf("relay_information.icon default = %q, want empty", got)
	}
}
