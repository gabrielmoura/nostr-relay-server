package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// LoadConfig carrega as configurações a partir do arquivo e define os padrões.
func LoadConfig() error {
	setDefaults(false)

	viper.SetConfigName("conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../..")
	viper.AddConfigPath("/etc/nrs")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return applyLoadedConfig()
}

func LoadConfigFromFile(path string) error {
	setDefaults(false)

	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return applyLoadedConfig()
}

func applyLoadedConfig() error {
	if err := validatePrivacyRequiredValue(); err != nil {
		return err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}
	applyStoreS3Environment(cfg)

	if cfg.AppEnv == "" {
		cfg.AppEnv = "production"
	}

	if cfg.DB.PostgresURI == "" {
		return fmt.Errorf("missing DB URI")
	}

	if err := cfg.normalizeRelayKeys(); err != nil {
		return err
	}
	if err := cfg.normalizeAdminKeys(); err != nil {
		return err
	}
	if err := cfg.ValidateAdminFeatures(); err != nil {
		return err
	}
	if err := cfg.ValidateNegentropyFeatures(); err != nil {
		return err
	}
	if err := cfg.ValidateMarmotFeatures(); err != nil {
		return err
	}
	if err := cfg.ValidateContentDefenseFeatures(); err != nil {
		return err
	}
	if err := cfg.ValidateStoreFeatures(); err != nil {
		return err
	}

	Cfg = cfg
	cfg.applySecurityRelayInformationDefaults()
	cfg.applyNIP11Capabilities()
	return nil
}

func validatePrivacyRequiredValue() error {
	if !viper.IsSet("privacy.required") {
		return nil
	}
	if _, ok := viper.Get("privacy.required").(bool); !ok {
		return fmt.Errorf("privacy.required must be a boolean")
	}
	return nil
}

func appendSupportedNIP(values []int, nip int) []int {
	for _, v := range values {
		if v == nip {
			return values
		}
	}
	return append(values, nip)
}

func removeSupportedNIP(values []int, nip int) []int {
	filtered := values[:0]
	for _, value := range values {
		if value != nip {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func (cfg *Config) applyNIP11Capabilities() {
	if cfg == nil {
		return
	}

	for _, nip := range []int{1, 9, 11, 40, 45} {
		cfg.RelayInformation.SupportedNIPs = appendSupportedNIP(cfg.RelayInformation.SupportedNIPs, nip)
	}

	cfg.setNIP11Capability(13, cfg.Relay.MinimumPOWLimit > 0)
	cfg.setNIP11Capability(29, cfg.NIP29.Enabled)
	cfg.setNIP11Capability(50, cfg.Search.Enabled)
	cfg.setNIP11Capability(42, cfg.Ws.AuthEnabled())
	cfg.setNIP11Capability(62, cfg.Relay.VanishEvent)
	cfg.setNIP11Capability(70, cfg.NIP70.Enabled)
	cfg.setNIP11Capability(77, cfg.EnableNegentropy)
	cfg.setNIP11Capability(86, cfg.NIP86Enabled())
	cfg.setNIP11Capability(96, cfg.Store.Enabled)
	cfg.setNIP11Capability(98, cfg.NIP86Enabled())
}

func (cfg *Config) setNIP11Capability(nip int, enabled bool) {
	if enabled {
		cfg.RelayInformation.SupportedNIPs = appendSupportedNIP(cfg.RelayInformation.SupportedNIPs, nip)
		return
	}
	cfg.RelayInformation.SupportedNIPs = removeSupportedNIP(cfg.RelayInformation.SupportedNIPs, nip)
}

// PrintYamlConfig exibe a configuração atual no formato YAML.
func PrintYamlConfig() {
	cfg, err := DefaultConfig()
	if err != nil {
		panic(err)
	}

	data, err := yaml.Marshal(cfg.Redacted())
	if err != nil {
		panic(err)
	}
	println(string(data))
}

func DefaultConfig() (*Config, error) {
	setDefaults(true)

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = "production"
	}
	applyStoreS3Environment(cfg)
	cfg.applySecurityRelayInformationDefaults()
	cfg.applyNIP11Capabilities()

	return cfg, nil
}

// WriteYamlConfig escreve a configuração atual em um arquivo YAML.
func WriteYamlConfig(filename string) error {
	cfg, err := DefaultConfig()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg.Redacted())
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return err
	}

	return nil
}

func applyStoreS3Environment(cfg *Config) {
	if cfg == nil {
		return
	}
	if value, ok := os.LookupEnv("NRS_STORE_S3_ACCESS_KEY"); ok {
		cfg.Store.S3.AccessKey = value
	}
	if value, ok := os.LookupEnv("NRS_STORE_S3_SECRET_KEY"); ok {
		cfg.Store.S3.SecretKey = value
	}
}

// Redacted returns a copy suitable for operator-facing configuration output.
func (cfg *Config) Redacted() *Config {
	if cfg == nil {
		return nil
	}

	redacted := *cfg
	redacted.Store.S3.AccessKey = ""
	redacted.Store.S3.SecretKey = ""
	return &redacted
}
