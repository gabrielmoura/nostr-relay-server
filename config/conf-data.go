package config

import (
	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	"github.com/goccy/go-json"
	"golang.org/x/time/rate"
	"net/url"
	"strings"
)

type Config struct {
	Port             int                      `json:"port" yaml:"port" mapstructure:"port"`
	AppEnv           string                   `json:"app_env" yaml:"app_env" mapstructure:"app_env"`
	Ws               WsConfig                 `json:"ws" yaml:"ws" mapstructure:"ws"`
	Anon             Anon                     `json:"anon" yaml:"anon" mapstructure:"anon"`
	RelayInformation RelayInformationDocument `json:"relay_information" yaml:"relay_information" mapstructure:"relay_information"`
	Relay            RelayConfig              `json:"relay" yaml:"relay" mapstructure:"relay"`
	DB               DbConfig                 `json:"db" yaml:"db" mapstructure:"db"`
	StreamUp         WsStreamConfig           `json:"stream_up" yaml:"stream_up" mapstructure:"stream_up"`
	StreamDown       WsStreamConfig           `json:"stream_down" yaml:"stream_down" mapstructure:"stream_down"`
	EnableNegentropy bool                     `json:"enable_negentropy" yaml:"enable_negentropy" mapstructure:"enable_negentropy"`
	Store            StoreConfig              `json:"store" yaml:"store" mapstructure:"store"`
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
	Relays  []string `json:"relays" yaml:"relays" mapstructure:"relays"`
	Enabled bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
}
type DbConfig struct {
	MaxConns    int32  `json:"max_conns" yaml:"max_conns" mapstructure:"max_conns"`
	MinConns    int32  `json:"min_conns" yaml:"min_conns" mapstructure:"min_conns"`
	PostgresURI string `json:"postgres_uri" yaml:"postgres_uri" mapstructure:"postgres_uri"`
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
}
type WsConfig struct {
	ReteLimit rate.Limit `json:"rate_limit" yaml:"rate_limit" mapstructure:"rate_limit"`
	Burst     int        `json:"burst" yaml:"burst" mapstructure:"burst"`
	Auth      bool       `json:"auth" yaml:"auth" mapstructure:"auth"`
}
type Anon struct {
	I2p       string `json:"i2p" yaml:"i2p" mapstructure:"i2p"`
	EnableI2p bool   `json:"enable_i2p" yaml:"enable_i2p" mapstructure:"enable_i2p"`
}

func (cfg *RelayInformationDocument) ToJson() (data []byte, err error) {
	data, err = json.Marshal(cfg)
	return
}

type RelayInformationDocument struct {
	URL string `json:"-" mapstructure:"url"`

	Name          string `json:"name,omitempty" mapstructure:"name"`
	Description   string `json:"description,omitempty" mapstructure:"description"`
	PubKey        string `json:"pub_key,omitempty" mapstructure:"pub_key"`
	PrivKey       string `mapstructure:"priv_key"`
	Contact       string `json:"contact,omitempty" mapstructure:"contact"`
	SupportedNIPs []int  `json:"supported_nips,omitempty" mapstructure:"supported_nips"`
	Software      string `json:"software,omitempty" mapstructure:"software"`
	Version       string `json:"version,omitempty" mapstructure:"version"`

	CanonicalURL string `json:"canonical_url,omitempty" yaml:"canonical_url" mapstructure:"canonical_url"`

	Limitation     *RelayLimitationDocument `json:"limitation,omitempty" mapstructure:"limitation,omitempty"`
	RelayCountries []string                 `json:"relay_countries,omitempty"`
	LanguageTags   []string                 `json:"language_tags,omitempty"`
	Tags           []string                 `json:"tags,omitempty"`
	PostingPolicy  string                   `json:"posting_policy,omitempty"`
	PaymentsURL    string                   `json:"payments_url,omitempty"`
	Fees           *RelayFeesDocument       `json:"fees,omitempty"`
	Icon           string                   `json:"icon" mapstructure:"icon"`
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
	MaxMessageLength int  `json:"max_message_length,omitempty"`
	MaxSubscriptions int  `json:"max_subscriptions,omitempty"`
	MaxFilters       int  `json:"max_filters,omitempty"`
	MaxLimit         int  `json:"max_limit,omitempty"`
	MaxSubidLength   int  `json:"max_subid_length,omitempty"`
	MaxEventTags     int  `json:"max_event_tags,omitempty"`
	MaxContentLength int  `json:"max_content_length,omitempty"`
	MinPowDifficulty int  `json:"min_pow_difficulty,omitempty"`
	AuthRequired     bool `json:"auth_required"`
	PaymentRequired  bool `json:"payment_required"`
	RestrictedWrites bool `json:"restricted_writes"`
}

type RelayFeesDocument struct {
	Admission []struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"admission,omitempty"`
	Subscription []struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
		Period int    `json:"period"`
	} `json:"subscription,omitempty"`
	Publication []struct {
		Kinds  []int  `json:"kinds"`
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"publication,omitempty"`
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
