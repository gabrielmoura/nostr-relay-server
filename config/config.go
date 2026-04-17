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
	viper.SetDefault("ws.auth", false)
	viper.SetDefault("ws.auth_mode", "none")
	viper.SetDefault("port", 9090)
	viper.SetDefault("admin_token", "")

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
	viper.SetDefault("relay.vanish_event", false)

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
	viper.SetDefault("db.max_conn_lifetime_minutes", 30)
	viper.SetDefault("db.max_conn_idle_minutes", 5)
	viper.SetDefault("db.health_check_period_seconds", 30)

	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.addr", "127.0.0.1:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.subscription_cleanup_interval_seconds", 60)
	viper.SetDefault("redis.subscription_stale_after_seconds", 120)
	viper.SetDefault("redis.cache.ban_ttl", 3600)
	viper.SetDefault("redis.cache.profile_ttl", 300)
	viper.SetDefault("redis.cache.query_ttl", 30)
	viper.SetDefault("redis.cache.query_meta_ttl", 30)
	viper.SetDefault("redis.cache.event_ttl", 600)
	viper.SetDefault("redis.cache.dedup_ttl", 3600)
	viper.SetDefault("redis.cache.nip05_doc_ttl", 86400)

	viper.SetDefault("ingestion.batch_size", 1000)
	viper.SetDefault("ingestion.batch_timeout_ms", 100)
	viper.SetDefault("ingestion.workers", 4)
	viper.SetDefault("ingestion.queue_size", 10000)

	viper.SetDefault("cron.enabled", true)
	viper.SetDefault("cron.db_optimization.enabled", false)
	viper.SetDefault("cron.db_optimization.schedule", "0 30 3 * * *")
	viper.SetDefault("cron.db_optimization.analyze", true)
	viper.SetDefault("cron.db_optimization.vacuum_analyze", false)
	viper.SetDefault("cron.db_optimization.reindex_event", false)

	viper.SetDefault("cron.reported_events_fetch.enabled", false)
	viper.SetDefault("cron.reported_events_fetch.schedule", "0 */30 * * * *")
	viper.SetDefault("cron.reported_events_fetch.relays", []string{})
	viper.SetDefault("cron.reported_events_fetch.lookback_hours", 24)
	viper.SetDefault("cron.reported_events_fetch.limit_per_relay", 200)

	viper.SetDefault("cron.delete_old_events.enabled", false)
	viper.SetDefault("cron.delete_old_events.schedule", "0 0 4 * * *")
	viper.SetDefault("cron.delete_old_events.older_than_days", 365)
	viper.SetDefault("cron.delete_old_events.batch_size", 2000)

	viper.SetDefault("cron.nip40.enabled", false)
	viper.SetDefault("cron.nip40.schedule", "0 */15 * * * *")
	viper.SetDefault("cron.nip40.batch_size", 2000)

	viper.SetDefault("stream.stream_up", true)
	viper.SetDefault("stream.stream_down", false)
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

	Cfg = cfg
	return nil
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
