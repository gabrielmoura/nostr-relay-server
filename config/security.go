package config

type SecurityConfig struct {
	Enabled   bool                    `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Whitelist SecurityWhitelistConfig `json:"whitelist" yaml:"whitelist" mapstructure:"whitelist"`
	Limits    SecurityLimitsConfig    `json:"limits" yaml:"limits" mapstructure:"limits"`
	Defense   SecurityDefenseConfig   `json:"defense" yaml:"defense" mapstructure:"defense"`
}

type SecurityWhitelistConfig struct {
	Enabled bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	PubKeys []string `json:"pubkeys" yaml:"pubkeys" mapstructure:"pubkeys"`
	IPs     []string `json:"ips" yaml:"ips" mapstructure:"ips"`
	CIDRs   []string `json:"cidrs" yaml:"cidrs" mapstructure:"cidrs"`
}

type SecurityLimitsConfig struct {
	MaxLimit            int `json:"max_limit" yaml:"max_limit" mapstructure:"max_limit"`
	MaxFiltersPerReq    int `json:"max_filters_per_req" yaml:"max_filters_per_req" mapstructure:"max_filters_per_req"`
	MaxMessageLength    int `json:"max_message_length" yaml:"max_message_length" mapstructure:"max_message_length"`
	MaxEventTags        int `json:"max_event_tags" yaml:"max_event_tags" mapstructure:"max_event_tags"`
	MaxContentLength    int `json:"max_content_length" yaml:"max_content_length" mapstructure:"max_content_length"`
	MaxConnectionsPerIP int `json:"max_connections_per_ip" yaml:"max_connections_per_ip" mapstructure:"max_connections_per_ip"`
}

type SecurityDefenseConfig struct {
	Enabled         bool                      `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	UseRedis        bool                      `json:"use_redis" yaml:"use_redis" mapstructure:"use_redis"`
	BlockTTLSeconds int                       `json:"block_ttl_seconds" yaml:"block_ttl_seconds" mapstructure:"block_ttl_seconds"`
	Event           SecurityWindowLimitConfig `json:"event" yaml:"event" mapstructure:"event"`
	Req             SecurityWindowLimitConfig `json:"req" yaml:"req" mapstructure:"req"`
}

type SecurityWindowLimitConfig struct {
	WindowSeconds  int `json:"window_seconds" yaml:"window_seconds" mapstructure:"window_seconds"`
	ThrottleAfter  int `json:"throttle_after" yaml:"throttle_after" mapstructure:"throttle_after"`
	RestrictAfter  int `json:"restrict_after" yaml:"restrict_after" mapstructure:"restrict_after"`
	TemporaryBlock int `json:"temporary_block_after" yaml:"temporary_block_after" mapstructure:"temporary_block_after"`
}

func (cfg *Config) applySecurityRelayInformationDefaults() {
	if cfg == nil || !cfg.Security.Enabled {
		return
	}

	if cfg.RelayInformation.Limitation == nil {
		cfg.RelayInformation.Limitation = &RelayLimitationDocument{}
	}

	setIntPtrIfNil(&cfg.RelayInformation.Limitation.MaxMessageLength, cfg.Security.Limits.MaxMessageLength)
	setIntPtrIfNil(&cfg.RelayInformation.Limitation.MaxFilters, cfg.Security.Limits.MaxFiltersPerReq)
	setIntPtrIfNil(&cfg.RelayInformation.Limitation.MaxLimit, cfg.Security.Limits.MaxLimit)
	setIntPtrIfNil(&cfg.RelayInformation.Limitation.MaxEventTags, cfg.Security.Limits.MaxEventTags)
	setIntPtrIfNil(&cfg.RelayInformation.Limitation.MaxContentLength, cfg.Security.Limits.MaxContentLength)
	setIntPtrIfNil(&cfg.RelayInformation.Limitation.MinPowDifficulty, cfg.Relay.MinimumPOWLimit)

	authRequired := cfg.Ws.RequireAuthForReq() || cfg.Ws.RequireAuthForEvent()
	setBoolPtrIfNil(&cfg.RelayInformation.Limitation.AuthRequired, authRequired)
}

func setIntPtrIfNil(target **int, value int) {
	if value <= 0 || *target != nil {
		return
	}
	v := value
	*target = &v
}

func setBoolPtrIfNil(target **bool, value bool) {
	if *target != nil {
		return
	}
	v := value
	*target = &v
}
