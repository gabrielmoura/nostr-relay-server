package config

import (
	"strings"

	"golang.org/x/time/rate"
)

type Config struct {
	Port             int                      `json:"port" yaml:"port" mapstructure:"port"`
	AppEnv           string                   `json:"app_env" yaml:"app_env" mapstructure:"app_env"`
	AdminToken       string                   `json:"admin_token" yaml:"admin_token" mapstructure:"admin_token"`
	AdminPubKey      string                   `json:"admin_pubkey" yaml:"admin_pubkey" mapstructure:"admin_pubkey"`
	Ws               WsConfig                 `json:"ws" yaml:"ws" mapstructure:"ws"`
	Security         SecurityConfig           `json:"security" yaml:"security" mapstructure:"security"`
	Anon             Anon                     `json:"anon" yaml:"anon" mapstructure:"anon"`
	RelayInformation RelayInformationDocument `json:"relay_information" yaml:"relay_information" mapstructure:"relay_information"`
	Relay            RelayConfig              `json:"relay" yaml:"relay" mapstructure:"relay"`
	DB               DbConfig                 `json:"db" yaml:"db" mapstructure:"db"`
	Redis            RedisConfig              `json:"redis" yaml:"redis" mapstructure:"redis"`
	Jobs             JobsConfig               `json:"jobs" yaml:"jobs" mapstructure:"jobs"`
	Ingestion        IngestionConfig          `json:"ingestion" yaml:"ingestion" mapstructure:"ingestion"`
	Cron             CronConfig               `json:"cron" yaml:"cron" mapstructure:"cron"`
	Stream           WsStreamConfig           `json:"stream" yaml:"stream" mapstructure:"stream"`
	EnableNegentropy bool                     `json:"enable_negentropy" yaml:"enable_negentropy" mapstructure:"enable_negentropy"`
	NIP86            NIP86Config              `json:"nip86" yaml:"nip86" mapstructure:"nip86"`
	Store            StoreConfig              `json:"store" yaml:"store" mapstructure:"store"`
	NIP29            NIP29Config              `json:"nip29" yaml:"nip29" mapstructure:"nip29"`
	WoT              WoTConfig                `json:"wot" yaml:"wot" mapstructure:"wot"`
}

type WoTConfig struct {
	Enabled              bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	RefreshIntervalHours int      `json:"refresh_interval_hours" yaml:"refresh_interval_hours" mapstructure:"refresh_interval_hours"`
	MinimumFollowers     int      `json:"minimum_followers" yaml:"minimum_followers" mapstructure:"minimum_followers"`
	TargetPubkey         string   `json:"target_pubkey" yaml:"target_pubkey" mapstructure:"target_pubkey"`
	MaxTrustNetwork      int      `json:"max_trust_network" yaml:"max_trust_network" mapstructure:"max_trust_network"`
	MaxOneHopNetwork     int      `json:"max_one_hop_network" yaml:"max_one_hop_network" mapstructure:"max_one_hop_network"`
	SeedRelays           []string `json:"seed_relays" yaml:"seed_relays" mapstructure:"seed_relays"`
	TrustedPubkeys       []string `json:"trusted_pubkeys" yaml:"trusted_pubkeys" mapstructure:"trusted_pubkeys"`
}

type StoreConfig struct {
	Enabled             bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	APIPath             string   `json:"api_path" yaml:"api_path" mapstructure:"api_path"`
	MediaPath           string   `json:"media_path" yaml:"media_path" mapstructure:"media_path"`
	AcceptedMimetypes   []string `json:"accepted_mimetypes" yaml:"accepted_mimetypes" mapstructure:"accepted_mimetypes"`
	AllowAdultContent   bool     `json:"allow_adult_content" yaml:"allow_adult_content" mapstructure:"allow_adult_content"`
	AllowViolentContent bool     `json:"allow_violent_content" yaml:"allow_violent_content" mapstructure:"allow_violent_content"`
	Names               []string `json:"names" yaml:"names" mapstructure:"names"`
}

type WsStreamConfig struct {
	Relays     []string `json:"relays" yaml:"relays" mapstructure:"relays"`
	StreamUp   bool     `json:"stream_up" yaml:"stream_up" mapstructure:"stream_up"`
	StreamDown bool     `json:"stream_down" yaml:"stream_down" mapstructure:"stream_down"`
}
type DbConfig struct {
	MaxConns                 int32  `json:"max_conns" yaml:"max_conns" mapstructure:"max_conns"`
	MinConns                 int32  `json:"min_conns" yaml:"min_conns" mapstructure:"min_conns"`
	PostgresURI              string `json:"postgres_uri" yaml:"postgres_uri" mapstructure:"postgres_uri"`
	MaxConnLifetimeMinutes   int32  `json:"max_conn_lifetime_minutes" yaml:"max_conn_lifetime_minutes" mapstructure:"max_conn_lifetime_minutes"`
	MaxConnIdleMinutes       int32  `json:"max_conn_idle_minutes" yaml:"max_conn_idle_minutes" mapstructure:"max_conn_idle_minutes"`
	HealthCheckPeriodSeconds int32  `json:"health_check_period_seconds" yaml:"health_check_period_seconds" mapstructure:"health_check_period_seconds"`
}
type RelayConfig struct {
	QueryLimit         int   `json:"query_limit" yaml:"query_limit" mapstructure:"query_limit"`
	QueryIDsLimit      int   `json:"query_ids_limit" yaml:"query_ids_limit" mapstructure:"query_ids_limit"`
	QueryAuthorsLimit  int   `json:"query_authors_limit" yaml:"query_authors_limit" mapstructure:"query_authors_limit"`
	QueryKindsLimit    int   `json:"query_kinds_limit" yaml:"query_kinds_limit" mapstructure:"query_kinds_limit"`
	QueryTagsLimit     int   `json:"query_tags_limit" yaml:"query_tags_limit" mapstructure:"query_tags_limit"`
	KeepRecentEvents   bool  `json:"keep_recent_events" yaml:"keep_recent_events" mapstructure:"keep_recent_events"`
	MaxEventSize       int   `json:"max_size_event_in_bytes" yaml:"max_size_event_in_bytes" mapstructure:"max_size_event_in_bytes"`
	FilterLimit        int   `json:"filter_limit" yaml:"filter_limit" mapstructure:"filter_limit"`
	ReportingLimit     int64 `json:"reporting_limit" yaml:"reporting_limit" mapstructure:"reporting_limit"`
	EnableAnonymousReq bool  `json:"enable_anonymous_req" yaml:"enable_anonymous_req" mapstructure:"enable_anonymous_req"`
	MaxTagValueLength  int   `json:"max_tag_value_length" yaml:"max_tag_value_length" mapstructure:"max_tag_value_length"`
	ProtectedKinds     []int `json:"-" yaml:"protected_kinds" mapstructure:"protected_kinds"`
	MinimumPOWLimit    int   `json:"-" yaml:"minimum_pow_limit" mapstructure:"minimum_pow_limit"`
	FakeDeletion       bool  `json:"fake_deletion" yaml:"fake_deletion" mapstructure:"fake_deletion"`
	VanishEvent        bool  `json:"vanish_event" yaml:"vanish_event" mapstructure:"vanish_event"`
	EnableEmptyFilter  bool  `json:"enable_empty_filter" yaml:"enable_empty_filter" mapstructure:"enable_empty_filter"`
	WhitelistKinds     []int `json:"whitelist_kinds" yaml:"whitelist_kinds" mapstructure:"whitelist_kinds"`
	BlacklistKinds     []int `json:"blacklist_kinds" yaml:"blacklist_kinds" mapstructure:"blacklist_kinds"`
}
type WsConfig struct {
	ReteLimit rate.Limit `json:"rate_limit" yaml:"rate_limit" mapstructure:"rate_limit"`
	Burst     int        `json:"burst" yaml:"burst" mapstructure:"burst"`
	Auth      bool       `json:"auth" yaml:"auth" mapstructure:"auth"`
	AuthMode  string     `json:"auth_mode" yaml:"auth_mode" mapstructure:"auth_mode"`
}

func (cfg WsConfig) NormalizedAuthMode() string {
	switch strings.ToLower(strings.TrimSpace(cfg.AuthMode)) {
	case "strict", "flexible", "optional", "none":
		return strings.ToLower(strings.TrimSpace(cfg.AuthMode))
	}
	if cfg.Auth {
		return "strict"
	}
	return "none"
}

func (cfg WsConfig) AuthEnabled() bool {
	return cfg.NormalizedAuthMode() != "none"
}

func (cfg WsConfig) RequireAuthForReq() bool {
	return cfg.NormalizedAuthMode() == "strict"
}

func (cfg WsConfig) RequireAuthForEvent() bool {
	mode := cfg.NormalizedAuthMode()
	return mode == "strict" || mode == "flexible"
}

type Anon struct {
	I2p       string `json:"i2p" yaml:"i2p" mapstructure:"i2p"`
	EnableI2p bool   `json:"enable_i2p" yaml:"enable_i2p" mapstructure:"enable_i2p"`
}

type RedisConfig struct {
	Enabled                            bool             `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Addr                               string           `json:"addr" yaml:"addr" mapstructure:"addr"`
	Password                           string           `json:"password" yaml:"password" mapstructure:"password"`
	DB                                 int              `json:"db" yaml:"db" mapstructure:"db"`
	PoolSize                           int              `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"`
	Queue                              RedisQueueConfig `json:"queue" yaml:"queue" mapstructure:"queue"`
	SubscriptionCleanupIntervalSeconds int              `json:"subscription_cleanup_interval_seconds" yaml:"subscription_cleanup_interval_seconds" mapstructure:"subscription_cleanup_interval_seconds"`
	SubscriptionStaleAfterSeconds      int              `json:"subscription_stale_after_seconds" yaml:"subscription_stale_after_seconds" mapstructure:"subscription_stale_after_seconds"`
	Cache                              CacheTTLConfig   `json:"cache" yaml:"cache" mapstructure:"cache"`
}

type RedisQueueConfig struct {
	Enabled             bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	ConsumerGroup       string `json:"consumer_group" yaml:"consumer_group" mapstructure:"consumer_group"`
	WorkerCount         int    `json:"worker_count" yaml:"worker_count" mapstructure:"worker_count"`
	BlockMS             int    `json:"block_ms" yaml:"block_ms" mapstructure:"block_ms"`
	BatchSize           int64  `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`
	MaxLenApprox        int64  `json:"max_len_approx" yaml:"max_len_approx" mapstructure:"max_len_approx"`
	BodyTTLSeconds      int    `json:"body_ttl_seconds" yaml:"body_ttl_seconds" mapstructure:"body_ttl_seconds"`
	ResultTTLSeconds    int    `json:"result_ttl_seconds" yaml:"result_ttl_seconds" mapstructure:"result_ttl_seconds"`
	MetricsTTLSeconds   int    `json:"metrics_ttl_seconds" yaml:"metrics_ttl_seconds" mapstructure:"metrics_ttl_seconds"`
	ReclaimIdleSeconds  int    `json:"reclaim_idle_seconds" yaml:"reclaim_idle_seconds" mapstructure:"reclaim_idle_seconds"`
	ReclaimBatchSize    int64  `json:"reclaim_batch_size" yaml:"reclaim_batch_size" mapstructure:"reclaim_batch_size"`
	PromoteBatchSize    int64  `json:"promote_batch_size" yaml:"promote_batch_size" mapstructure:"promote_batch_size"`
	PromoteIntervalMS   int    `json:"promote_interval_ms" yaml:"promote_interval_ms" mapstructure:"promote_interval_ms"`
	RecentJobsListLimit int64  `json:"recent_jobs_list_limit" yaml:"recent_jobs_list_limit" mapstructure:"recent_jobs_list_limit"`
}

type JobsConfig struct {
	Enabled                   bool                  `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	DefaultQueue              string                `json:"default_queue" yaml:"default_queue" mapstructure:"default_queue"`
	DefaultTimeoutSeconds     int                   `json:"default_timeout_seconds" yaml:"default_timeout_seconds" mapstructure:"default_timeout_seconds"`
	DefaultMaxAttempts        int                   `json:"default_max_attempts" yaml:"default_max_attempts" mapstructure:"default_max_attempts"`
	RetryBaseDelayMS          int                   `json:"retry_base_delay_ms" yaml:"retry_base_delay_ms" mapstructure:"retry_base_delay_ms"`
	RetryMaxDelayMS           int                   `json:"retry_max_delay_ms" yaml:"retry_max_delay_ms" mapstructure:"retry_max_delay_ms"`
	RetryJitterMS             int                   `json:"retry_jitter_ms" yaml:"retry_jitter_ms" mapstructure:"retry_jitter_ms"`
	ActiveWorkerWindowSeconds int                   `json:"active_worker_window_seconds" yaml:"active_worker_window_seconds" mapstructure:"active_worker_window_seconds"`
	Download                  JobsTargetQueueConfig `json:"download" yaml:"download" mapstructure:"download"`
	Sync                      JobsTargetQueueConfig `json:"sync" yaml:"sync" mapstructure:"sync"`
	Cron                      JobsTargetQueueConfig `json:"cron" yaml:"cron" mapstructure:"cron"`
}

type JobsTargetQueueConfig struct {
	Queue    string `json:"queue" yaml:"queue" mapstructure:"queue"`
	Priority string `json:"priority" yaml:"priority" mapstructure:"priority"`
}

type CacheTTLConfig struct {
	BanTTL       int `json:"ban_ttl" yaml:"ban_ttl" mapstructure:"ban_ttl"`
	ProfileTTL   int `json:"profile_ttl" yaml:"profile_ttl" mapstructure:"profile_ttl"`
	QueryTTL     int `json:"query_ttl" yaml:"query_ttl" mapstructure:"query_ttl"`
	QueryMetaTTL int `json:"query_meta_ttl" yaml:"query_meta_ttl" mapstructure:"query_meta_ttl"`
	EventTTL     int `json:"event_ttl" yaml:"event_ttl" mapstructure:"event_ttl"`
	DedupTTL     int `json:"dedup_ttl" yaml:"dedup_ttl" mapstructure:"dedup_ttl"`
	NIP05DocTTL  int `json:"nip05_doc_ttl" yaml:"nip05_doc_ttl" mapstructure:"nip05_doc_ttl"`
}

type IngestionConfig struct {
	BatchSize      int `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`
	BatchTimeoutMs int `json:"batch_timeout_ms" yaml:"batch_timeout_ms" mapstructure:"batch_timeout_ms"`
	Workers        int `json:"workers" yaml:"workers" mapstructure:"workers"`
	QueueSize      int `json:"queue_size" yaml:"queue_size" mapstructure:"queue_size"`
}

type CronConfig struct {
	Enabled             bool                      `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	DBOptimization      CronDBOptimizationConfig  `json:"db_optimization" yaml:"db_optimization" mapstructure:"db_optimization"`
	ReportedEventsFetch CronReportedEventsConfig  `json:"reported_events_fetch" yaml:"reported_events_fetch" mapstructure:"reported_events_fetch"`
	DeleteOldEvents     CronDeleteOldEventsConfig `json:"delete_old_events" yaml:"delete_old_events" mapstructure:"delete_old_events"`
	NIP40               CronNIP40Config           `json:"nip40" yaml:"nip40" mapstructure:"nip40"`
}

type CronDBOptimizationConfig struct {
	Enabled       bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Schedule      string `json:"schedule" yaml:"schedule" mapstructure:"schedule"`
	Analyze       bool   `json:"analyze" yaml:"analyze" mapstructure:"analyze"`
	VacuumAnalyze bool   `json:"vacuum_analyze" yaml:"vacuum_analyze" mapstructure:"vacuum_analyze"`
	ReindexEvent  bool   `json:"reindex_event" yaml:"reindex_event" mapstructure:"reindex_event"`
}

type CronReportedEventsConfig struct {
	Enabled       bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Schedule      string   `json:"schedule" yaml:"schedule" mapstructure:"schedule"`
	Relays        []string `json:"relays" yaml:"relays" mapstructure:"relays"`
	LookbackHours int      `json:"lookback_hours" yaml:"lookback_hours" mapstructure:"lookback_hours"`
	LimitPerRelay int      `json:"limit_per_relay" yaml:"limit_per_relay" mapstructure:"limit_per_relay"`
}

type CronDeleteOldEventsConfig struct {
	Enabled       bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Schedule      string `json:"schedule" yaml:"schedule" mapstructure:"schedule"`
	OlderThanDays int    `json:"older_than_days" yaml:"older_than_days" mapstructure:"older_than_days"`
	BatchSize     int    `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`
}

type CronNIP40Config struct {
	Enabled   bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Schedule  string `json:"schedule" yaml:"schedule" mapstructure:"schedule"`
	BatchSize int    `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`
}

type NIP29Config struct {
	Enabled                   bool                   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	RelayScope                string                 `json:"relay_scope" yaml:"relay_scope" mapstructure:"relay_scope"`
	CacheTTLSeconds           int                    `json:"cache_ttl_seconds" yaml:"cache_ttl_seconds" mapstructure:"cache_ttl_seconds"`
	MembershipCacheTTLSeconds int                    `json:"membership_cache_ttl_seconds" yaml:"membership_cache_ttl_seconds" mapstructure:"membership_cache_ttl_seconds"`
	BanCacheTTLSeconds        int                    `json:"ban_cache_ttl_seconds" yaml:"ban_cache_ttl_seconds" mapstructure:"ban_cache_ttl_seconds"`
	TimelineCacheTTLSeconds   int                    `json:"timeline_cache_ttl_seconds" yaml:"timeline_cache_ttl_seconds" mapstructure:"timeline_cache_ttl_seconds"`
	GroupCreatorRole          string                 `json:"group_creator_role" yaml:"group_creator_role" mapstructure:"group_creator_role"`
	DefaultRoles              []NIP29RoleConfig      `json:"default_roles" yaml:"default_roles" mapstructure:"default_roles"`
	Create                    NIP29CreateConfig      `json:"create" yaml:"create" mapstructure:"create"`
	Moderation                NIP29ModerationConfig  `json:"moderation" yaml:"moderation" mapstructure:"moderation"`
	Admission                 NIP29AdmissionConfig   `json:"admission" yaml:"admission" mapstructure:"admission"`
	Invite                    NIP29InviteConfig      `json:"invite" yaml:"invite" mapstructure:"invite"`
	PoW                       NIP29PoWConfig         `json:"pow" yaml:"pow" mapstructure:"pow"`
	Timeline                  NIP29TimelineConfig    `json:"timeline" yaml:"timeline" mapstructure:"timeline"`
	Advanced                  NIP29AdvancedConfig    `json:"advanced" yaml:"advanced" mapstructure:"advanced"`
	Permissions               NIP29PermissionToggles `json:"permissions" yaml:"permissions" mapstructure:"permissions"`
}

type NIP29RoleConfig struct {
	Name        string   `json:"name" yaml:"name" mapstructure:"name"`
	Description string   `json:"description" yaml:"description" mapstructure:"description"`
	Permissions []string `json:"permissions" yaml:"permissions" mapstructure:"permissions"`
}

type NIP29CreateConfig struct {
	Enabled            bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	MaxGroupsPerPubkey int  `json:"max_groups_per_pubkey" yaml:"max_groups_per_pubkey" mapstructure:"max_groups_per_pubkey"`
}

type NIP29ModerationConfig struct {
	AllowPrivateGroups      bool `json:"allow_private_groups" yaml:"allow_private_groups" mapstructure:"allow_private_groups"`
	RequireRecentModeration bool `json:"require_recent_moderation" yaml:"require_recent_moderation" mapstructure:"require_recent_moderation"`
	RecentWindowSeconds     int  `json:"recent_window_seconds" yaml:"recent_window_seconds" mapstructure:"recent_window_seconds"`
}

type NIP29AdmissionConfig struct {
	DefaultClosed             bool `json:"default_closed" yaml:"default_closed" mapstructure:"default_closed"`
	DefaultPrivate            bool `json:"default_private" yaml:"default_private" mapstructure:"default_private"`
	DefaultRestricted         bool `json:"default_restricted" yaml:"default_restricted" mapstructure:"default_restricted"`
	DefaultHidden             bool `json:"default_hidden" yaml:"default_hidden" mapstructure:"default_hidden"`
	RequireMembershipForWrite bool `json:"require_membership_for_write" yaml:"require_membership_for_write" mapstructure:"require_membership_for_write"`
	AllowLatePublication      bool `json:"allow_late_publication" yaml:"allow_late_publication" mapstructure:"allow_late_publication"`
}

type NIP29InviteConfig struct {
	Enabled           bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	DefaultMaxUses    int  `json:"default_max_uses" yaml:"default_max_uses" mapstructure:"default_max_uses"`
	DefaultTTLSeconds int  `json:"default_ttl_seconds" yaml:"default_ttl_seconds" mapstructure:"default_ttl_seconds"`
	AllowMultiUse     bool `json:"allow_multi_use" yaml:"allow_multi_use" mapstructure:"allow_multi_use"`
}

type NIP29PoWConfig struct {
	Enabled                 bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	DefaultMinDifficulty    int  `json:"default_min_difficulty" yaml:"default_min_difficulty" mapstructure:"default_min_difficulty"`
	ModerationMinDifficulty int  `json:"moderation_min_difficulty" yaml:"moderation_min_difficulty" mapstructure:"moderation_min_difficulty"`
}

type NIP29TimelineConfig struct {
	Enabled              bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	RequiredOnModeration bool `json:"required_on_moderation" yaml:"required_on_moderation" mapstructure:"required_on_moderation"`
	MinReferences        int  `json:"min_references" yaml:"min_references" mapstructure:"min_references"`
	RecentWindow         int  `json:"recent_window" yaml:"recent_window" mapstructure:"recent_window"`
}

type NIP29AdvancedConfig struct {
	EmitMemberListEvents  bool `json:"emit_member_list_events" yaml:"emit_member_list_events" mapstructure:"emit_member_list_events"`
	EmitRoleEvents        bool `json:"emit_role_events" yaml:"emit_role_events" mapstructure:"emit_role_events"`
	CacheMembershipLookup bool `json:"cache_membership_lookup" yaml:"cache_membership_lookup" mapstructure:"cache_membership_lookup"`
	CacheGroupMetadata    bool `json:"cache_group_metadata" yaml:"cache_group_metadata" mapstructure:"cache_group_metadata"`
}

type NIP29PermissionToggles struct {
	CreateInvite bool `json:"create_invite" yaml:"create_invite" mapstructure:"create_invite"`
}

type NIP86Config struct {
	Enabled           bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	AuthWindowSeconds int  `json:"auth_window_seconds" yaml:"auth_window_seconds" mapstructure:"auth_window_seconds"`
	CacheTTLSeconds   int  `json:"cache_ttl_seconds" yaml:"cache_ttl_seconds" mapstructure:"cache_ttl_seconds"`
}

var Cfg *Config
