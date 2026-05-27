package config

import (
	"fmt"

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

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

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

	Cfg = cfg
	cfg.applySecurityRelayInformationDefaults()
	if cfg.NIP29.Enabled {
		cfg.RelayInformation.SupportedNIPs = appendSupportedNIP(cfg.RelayInformation.SupportedNIPs, 29)
	}
	if cfg.NIP86Enabled() {
		cfg.RelayInformation.SupportedNIPs = appendSupportedNIP(cfg.RelayInformation.SupportedNIPs, 86)
	}
	if cfg.Ws.AuthEnabled() {
		cfg.RelayInformation.SupportedNIPs = appendSupportedNIP(cfg.RelayInformation.SupportedNIPs, 42)
	}
	if cfg.EnableNegentropy {
		cfg.RelayInformation.SupportedNIPs = appendSupportedNIP(cfg.RelayInformation.SupportedNIPs, 77)
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

// PrintYamlConfig exibe a configuração atual no formato YAML.
func PrintYamlConfig() {
	cfg, err := DefaultConfig()
	if err != nil {
		panic(err)
	}

	data, err := yaml.Marshal(cfg)
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

	return cfg, nil
}

// WriteYamlConfig escreve a configuração atual em um arquivo YAML.
func WriteYamlConfig(filename string) error {
	setDefaults(true)

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = "production"
	}

	if err := viper.WriteConfigAs(filename); err != nil {
		return err
	}
	return nil
}
