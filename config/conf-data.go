package config

import (
	"github.com/goccy/go-json"
	"golang.org/x/time/rate"
	"slices"
)

type Config struct {
	Port             int                      `json:"port" yaml:"port" mapstructure:"port"`
	AppEnv           string                   `json:"app_env" yaml:"app_env" mapstructure:"app_env"`
	Ws               WsConfig                 `json:"ws" yaml:"ws" mapstructure:"ws"`
	Anon             Anon                     `json:"anon" yaml:"anon" mapstructure:"anon"`
	RelayInformation RelayInformationDocument `json:"relay_information" yaml:"relay_information" mapstructure:"relay_information"`
	Relay            RelayConfig              `json:"relay" yaml:"relay" mapstructure:"relay"`
	DB               DbConfig                 `json:"db" yaml:"db" mapstructure:"db"`
}
type DbConfig struct {
	MaxConns    int32  `json:"max_conns" yaml:"max_conns" mapstructure:"max_conns"`
	MinConns    int32  `json:"min_conns" yaml:"min_conns" mapstructure:"min_conns"`
	PostgresURI string `json:"postgres_uri" yaml:"postgres_uri" mapstructure:"postgres_uri"`
}
type RelayConfig struct {
	QueryLimit        int  `json:"query_limit" yaml:"query_limit" mapstructure:"query_limit"`
	QueryIDsLimit     int  `json:"query_ids_limit" yaml:"query_ids_limit" mapstructure:"query_ids_limit"`
	QueryAuthorsLimit int  `json:"query_authors_limit" yaml:"query_authors_limit" mapstructure:"query_authors_limit"`
	QueryKindsLimit   int  `json:"query_kinds_limit" yaml:"query_kinds_limit" mapstructure:"query_kinds_limit"`
	QueryTagsLimit    int  `json:"query_tags_limit" yaml:"query_tags_limit" mapstructure:"query_tags_limit"`
	KeepRecentEvents  bool `json:"keep_recent_events" yaml:"keep_recent_events" mapstructure:"keep_recent_events"`
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
	URL string `json:"-"`

	Name          string `json:"name" mapstructure:"name"`
	Description   string `json:"description" mapstructure:"description"`
	PubKey        string `json:"pubkey" mapstructure:"pubkey"`
	Contact       string `json:"contact" mapstructure:"contact"`
	SupportedNIPs []any  `json:"supported_nips" mapstructure:"supported_nips"`
	Software      string `json:"software" mapstructure:"software"`
	Version       string `json:"version" mapstructure:"version"`

	CanonicalURL string `json:"canonical_url" yaml:"canonical_url" mapstructure:"canonical_url"`

	Limitation     *RelayLimitationDocument `json:"limitation,omitempty" mapstructure:"limitation,omitempty"`
	RelayCountries []string                 `json:"relay_countries,omitempty"`
	LanguageTags   []string                 `json:"language_tags,omitempty"`
	Tags           []string                 `json:"tags,omitempty"`
	PostingPolicy  string                   `json:"posting_policy,omitempty"`
	PaymentsURL    string                   `json:"payments_url,omitempty"`
	Fees           *RelayFeesDocument       `json:"fees,omitempty"`
	Icon           string                   `json:"icon" mapstructure:"icon"`
}

func (info *RelayInformationDocument) AddSupportedNIP(number int) {
	idx := slices.IndexFunc(info.SupportedNIPs, func(n any) bool { return n == number })
	if idx != -1 {
		return
	}

	info.SupportedNIPs = append(info.SupportedNIPs, number)
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
