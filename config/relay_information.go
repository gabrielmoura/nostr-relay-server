package config

import (
	"net/url"
	"strings"

	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

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
	CanonicalURL   string `json:"canonical_url,omitempty" yaml:"canonical_url" mapstructure:"canonical_url"`

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
	if cfg == nil {
		return errs
	}

	if canonicalURL := strings.TrimSpace(cfg.CanonicalURL); canonicalURL != "" {
		url1, err := url.Parse(canonicalURL)
		if err != nil || url1 == nil || (url1.Scheme != "ws" && url1.Scheme != "wss") {
			errs = append(errs, errors2.ErrInvalidCanonicalURL)
		}
	}

	if relayURL := strings.TrimSpace(cfg.URL); relayURL != "" {
		url2, err := url.Parse(relayURL)
		if err != nil || url2 == nil || !strings.Contains(url2.Scheme, "http") {
			errs = append(errs, errors2.ErrInvalidURL)
		}
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

type FileServerConfig struct {
	APIURL         string          `json:"api_url"`
	DownloadURL    string          `json:"download_url,omitempty"`
	DelegatedToURL string          `json:"delegated_to_url,omitempty"`
	SupportedNIPS  []int           `json:"supported_nips,omitempty"`
	TOSURL         string          `json:"tos_url,omitempty"`
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
