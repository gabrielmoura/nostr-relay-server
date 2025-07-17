package config

import (
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// setDefaults configura os valores padrão para a aplicação.
func setDefaults(export bool) {
	viper.SetDefault("ws.burst", 5)
	viper.SetDefault("ws.rate_limit", 1)
	viper.SetDefault("port", 9090)

	viper.SetDefault("relay_information.name", "Nostr Relay Server")
	viper.SetDefault("relay_information.description", "A Nostr Relay Server")
	viper.SetDefault("relay_information.supported_nips", []int{11, 1, 2, 4, 25})
	viper.SetDefault("relay_information.software", "https://github.com/gabrielmoura/nostr-relay-server")
	viper.SetDefault("relay_information.version", "0.1.0")
	viper.SetDefault("relay_information.icon", fmt.Sprintf("http://localhost:%s/nostr.png", viper.GetString("port")))
	//canonical_url
	viper.SetDefault("relay_information.canonical_url", fmt.Sprintf("ws://localhost:%s/relay", viper.GetString("port")))
	viper.SetDefault("relay_information.url", fmt.Sprintf("http://localhost:%s", viper.GetString("port")))

	viper.SetDefault("relay.query_limit", 100)
	viper.SetDefault("relay.query_ids_limit", 500)
	viper.SetDefault("relay.query_authors_limit", 500)
	viper.SetDefault("relay.query_kinds_limit", 10)
	viper.SetDefault("relay.query_tags_limit", 100)
	viper.SetDefault("relay.max_tag_value_length", 100)
	viper.SetDefault("relay.keep_recent_events", true)
	viper.SetDefault("relay.max_size_event_in_bytes", 100000) // 100KB
	viper.SetDefault("relay.filter_limit", 99999999)
	viper.SetDefault("relay.reporting_limit", 5) // 5 reports to ban a user
	viper.SetDefault("relay.enable_anonymous_req", true)
	viper.SetDefault("relay.fake_deletion", false)

	viper.SetDefault("store.api_path", fmt.Sprintf("http://localhost:%s/upload", viper.GetString("port")))
	viper.SetDefault("store.media_path", fmt.Sprintf("http://localhost:%s/blob", viper.GetString("port")))
	viper.SetDefault("store.enabled", false)
	viper.SetDefault("store.accepted_mimetypes", []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/svg+xml",
		"video/mp4",
		"video/webm",
		"video/ogg",
		"audio/mpeg",
		"audio/ogg",
		"audio/wav",
		"audio/flac",
		"audio/aac",
		"audio/mp4",
		"audio/opus",
		"audio/vorbis",
	},
	)

	viper.SetDefault("db.max_conns", 10)
	viper.SetDefault("db.min_conns", 1)

	viper.SetDefault("stream_up.enabled", true)
	viper.SetDefault("stream_down.enabled", false)
	viper.SetDefault("enable_negentropy", false)
	viper.SetDefault("relay.protected_kinds", []int{nostr.KindApplicationSpecificData, nostr.KindEncryptedDirectMessage})
	viper.SetDefault("relay.minimum_pow_limit", 0)

	if export {
		viper.SetDefault("db.postgres_uri", "postgres://user:password@localhost:5432/dbname")
	}
}

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

	Cfg = cfg
	return nil
}

// PrintYamlConfig exibe a configuração atual no formato YAML.
func PrintYamlConfig() {
	setDefaults(true)

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		panic(err)
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = "production"
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	println(string(data))
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
