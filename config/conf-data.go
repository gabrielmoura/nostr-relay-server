package config

import (
	"net/url"
	"strings"

	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"golang.org/x/time/rate"
)

type Config struct {
	Port             int                      `json:"port" yaml:"port" mapstructure:"port"`
	AppEnv           string                   `json:"app_env" yaml:"app_env" mapstructure:"app_env"`
	AdminToken       string                   `json:"admin_token" yaml:"admin_token" mapstructure:"admin_token"`
	Ws               WsConfig                 `json:"ws" yaml:"ws" mapstructure:"ws"`
	Security         SecurityConfig           `json:"security" yaml:"security" mapstructure:"security"`
	Anon             Anon                     `json:"anon" yaml:"anon" mapstructure:"anon"`
	RelayInformation RelayInformationDocument `json:"relay_information" yaml:"relay_information" mapstructure:"relay_information"`
	Relay            RelayConfig              `json:"relay" yaml:"relay" mapstructure:"relay"`
	DB               DbConfig                 `json:"db" yaml:"db" mapstructure:"db"`
	Redis            RedisConfig              `json:"redis" yaml:"redis" mapstructure:"redis"`
	Ingestion        IngestionConfig          `json:"ingestion" yaml:"ingestion" mapstructure:"ingestion"`
	Cron             CronConfig               `json:"cron" yaml:"cron" mapstructure:"cron"`
	Stream           WsStreamConfig           `json:"stream" yaml:"stream" mapstructure:"stream"`
	EnableNegentropy bool                     `json:"enable_negentropy" yaml:"enable_negentropy" mapstructure:"enable_negentropy"`
	Store            StoreConfig              `json:"store" yaml:"store" mapstructure:"store"`
	NIP29            NIP29Config              `json:"nip29" yaml:"nip29" mapstructure:"nip29"`
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
	Enabled                            bool           `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Addr                               string         `json:"addr" yaml:"addr" mapstructure:"addr"`
	Password                           string         `json:"password" yaml:"password" mapstructure:"password"`
	DB                                 int            `json:"db" yaml:"db" mapstructure:"db"`
	PoolSize                           int            `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"`
	SubscriptionCleanupIntervalSeconds int            `json:"subscription_cleanup_interval_seconds" yaml:"subscription_cleanup_interval_seconds" mapstructure:"subscription_cleanup_interval_seconds"`
	SubscriptionStaleAfterSeconds      int            `json:"subscription_stale_after_seconds" yaml:"subscription_stale_after_seconds" mapstructure:"subscription_stale_after_seconds"`
	Cache                              CacheTTLConfig `json:"cache" yaml:"cache" mapstructure:"cache"`
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

func (cfg *RelayInformationDocument) ToJson() (data []byte, err error) {
	data, err = json.Marshal(cfg)
	return
}

type RelayInformationDocument struct {
	URL string `json:"-" yaml:"url" mapstructure:"url"`

	Name           string `json:"name,omitempty" yaml:"name" mapstructure:"name"`
	Description    string `json:"description,omitempty" yaml:"description" mapstructure:"description"`
	Banner         string `json:"banner,omitempty" yaml:"banner" mapstructure:"banner"`
	PubKey         string `json:"pubkey,omitempty" yaml:"pub_key" mapstructure:"pub_key"`
	Self           string `json:"self,omitempty" yaml:"self" mapstructure:"self"`
	PrivKey        string `json:"-" yaml:"priv_key" mapstructure:"priv_key"`
	Contact        string `json:"contact,omitempty" yaml:"contact" mapstructure:"contact"`
	SupportedNIPs  []int  `json:"supported_nips,omitempty" yaml:"supported_nips" mapstructure:"supported_nips"`
	Software       string `json:"software,omitempty" yaml:"software" mapstructure:"software"`
	Version        string `json:"version,omitempty" yaml:"version" mapstructure:"version"`
	TermsOfService string `json:"terms_of_service,omitempty" yaml:"terms_of_service" mapstructure:"terms_of_service"`

	CanonicalURL string `json:"canonical_url,omitempty" yaml:"canonical_url" mapstructure:"canonical_url"`

	Limitation     *RelayLimitationDocument `json:"limitation,omitempty" yaml:"limitation" mapstructure:"limitation"`
	RelayCountries []string                 `json:"relay_countries,omitempty" yaml:"relay_countries" mapstructure:"relay_countries"`
	LanguageTags   []string                 `json:"language_tags,omitempty" yaml:"language_tags" mapstructure:"language_tags"`
	Tags           []string                 `json:"tags,omitempty" yaml:"tags" mapstructure:"tags"`
	PostingPolicy  string                   `json:"posting_policy,omitempty" yaml:"posting_policy" mapstructure:"posting_policy"`
	PaymentsURL    string                   `json:"payments_url,omitempty" yaml:"payments_url" mapstructure:"payments_url"`
	Fees           *RelayFeesDocument       `json:"fees,omitempty" yaml:"fees" mapstructure:"fees"`
	Icon           string                   `json:"icon,omitempty" yaml:"icon" mapstructure:"icon"`
}

func (cfg *RelayInformationDocument) GetPrivKey() string {
	return cfg.PrivKey
}
func (cfg *RelayInformationDocument) SetPrivKey(privKey string) {
	cfg.PrivKey = privKey
}

func (cfg *RelayInformationDocument) Check() []error {
	var errs []error
	url1, _ := url.Parse(cfg.CanonicalURL)
	if url1.Scheme != "wss" {
		errs = append(errs, errors2.ErrInvalidCanonicalURL)
	}
	url2, _ := url.Parse(cfg.URL)
	if strings.Contains(url2.Scheme, "http") {
		errs = append(errs, errors2.ErrInvalidURL)
	}

	return errs
}

type RelayLimitationDocument struct {
	MaxMessageLength    *int  `json:"max_message_length,omitempty" yaml:"max_message_length" mapstructure:"max_message_length"`
	MaxSubscriptions    *int  `json:"max_subscriptions,omitempty" yaml:"max_subscriptions" mapstructure:"max_subscriptions"`
	MaxFilters          *int  `json:"max_filters,omitempty" yaml:"max_filters" mapstructure:"max_filters"`
	MaxLimit            *int  `json:"max_limit,omitempty" yaml:"max_limit" mapstructure:"max_limit"`
	DefaultLimit        *int  `json:"default_limit,omitempty" yaml:"default_limit" mapstructure:"default_limit"`
	MaxSubidLength      *int  `json:"max_subid_length,omitempty" yaml:"max_subid_length" mapstructure:"max_subid_length"`
	MaxEventTags        *int  `json:"max_event_tags,omitempty" yaml:"max_event_tags" mapstructure:"max_event_tags"`
	MaxContentLength    *int  `json:"max_content_length,omitempty" yaml:"max_content_length" mapstructure:"max_content_length"`
	MinPowDifficulty    *int  `json:"min_pow_difficulty,omitempty" yaml:"min_pow_difficulty" mapstructure:"min_pow_difficulty"`
	CreatedAtLowerLimit *int  `json:"created_at_lower_limit,omitempty" yaml:"created_at_lower_limit" mapstructure:"created_at_lower_limit"`
	CreatedAtUpperLimit *int  `json:"created_at_upper_limit,omitempty" yaml:"created_at_upper_limit" mapstructure:"created_at_upper_limit"`
	AuthRequired        *bool `json:"auth_required,omitempty" yaml:"auth_required" mapstructure:"auth_required"`
	PaymentRequired     *bool `json:"payment_required,omitempty" yaml:"payment_required" mapstructure:"payment_required"`
	RestrictedWrites    *bool `json:"restricted_writes,omitempty" yaml:"restricted_writes" mapstructure:"restricted_writes"`
}

type RelayFeesDocument struct {
	Admission []struct {
		Amount int    `json:"amount" yaml:"amount" mapstructure:"amount"`
		Unit   string `json:"unit" yaml:"unit" mapstructure:"unit"`
	} `json:"admission,omitempty" yaml:"admission" mapstructure:"admission"`
	Subscription []struct {
		Amount int    `json:"amount" yaml:"amount" mapstructure:"amount"`
		Unit   string `json:"unit" yaml:"unit" mapstructure:"unit"`
		Period int    `json:"period" yaml:"period" mapstructure:"period"`
	} `json:"subscription,omitempty" yaml:"subscription" mapstructure:"subscription"`
	Publication []struct {
		Kinds  []int  `json:"kinds" yaml:"kinds" mapstructure:"kinds"`
		Amount int    `json:"amount" yaml:"amount" mapstructure:"amount"`
		Unit   string `json:"unit" yaml:"unit" mapstructure:"unit"`
	} `json:"publication,omitempty" yaml:"publication" mapstructure:"publication"`
}

var Cfg *Config

type FileServerConfig struct {
	APIURL         string          `json:"api_url"`
	DownloadURL    string          `json:"download_url,omitempty"`
	DelegatedToURL string          `json:"delegated_to_url,omitempty"`
	SupportedNIPS  []int           `json:"supported_nips,omitempty"`
	TOSURL         string          `json:"tos_url,omitempty"` // Terms of Service URL
	ContentTypes   []string        `json:"content_types,omitempty"`
	Plans          map[string]Plan `json:"plans,omitempty"`
}

type Plan struct {
	Name                 string   `json:"name"`
	IsNIP98Required      bool     `json:"is_nip98_required"`
	URL                  string   `json:"url,omitempty"`
	MaxByteSize          int64    `json:"max_byte_size"`
	FileExpiration       [2]int   `json:"file_expiration"`
	MediaTransformations []string `json:"media_transformations,omitempty"`
}
