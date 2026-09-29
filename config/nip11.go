package config

import (
	"net/url"
	"strings"
)

type publicRelayInformationDocument struct {
	Name                   string                   `json:"name,omitempty"`
	Description            string                   `json:"description,omitempty"`
	Banner                 string                   `json:"banner,omitempty"`
	Icon                   string                   `json:"icon,omitempty"`
	PubKey                 string                   `json:"pubkey,omitempty"`
	Self                   string                   `json:"self,omitempty"`
	Contact                string                   `json:"contact,omitempty"`
	SupportedNIPs          []int                    `json:"supported_nips,omitempty"`
	Software               string                   `json:"software,omitempty"`
	Version                string                   `json:"version,omitempty"`
	TermsOfService         string                   `json:"terms_of_service,omitempty"`
	Limitation             *RelayLimitationDocument `json:"limitation,omitempty"`
	RelayCountries         []string                 `json:"relay_countries,omitempty"`
	LanguageTags           []string                 `json:"language_tags,omitempty"`
	Tags                   []string                 `json:"tags,omitempty"`
	PostingPolicy          string                   `json:"posting_policy,omitempty"`
	PrivacyPolicy          string                   `json:"privacy_policy,omitempty"`
	PaymentsURL            string                   `json:"payments_url,omitempty"`
	Fees                   *RelayFeesDocument       `json:"fees,omitempty"`
	Retention              []RelayRetentionDocument `json:"retention,omitempty"`
	NIP50                  []string                 `json:"nip50,omitempty"`
	SupportedNIPExtensions []string                 `json:"supported_nip_extensions,omitempty"`
	SupportedGRASPs        []string                 `json:"supported_grasps,omitempty"`
}

func (cfg *RelayInformationDocument) PublicNIP11() any {
	if cfg == nil {
		return publicRelayInformationDocument{}
	}

	doc := publicRelayInformationDocument{
		Name:                   cfg.Name,
		Description:            cfg.Description,
		Banner:                 cfg.Banner,
		Icon:                   cfg.EffectiveIcon(),
		PubKey:                 cfg.PubKey,
		Self:                   cfg.Self,
		Contact:                cfg.Contact,
		SupportedNIPs:          cfg.SupportedNIPs,
		Software:               cfg.Software,
		Version:                cfg.Version,
		TermsOfService:         cfg.TermsOfService,
		RelayCountries:         cfg.RelayCountries,
		LanguageTags:           cfg.LanguageTags,
		Tags:                   cfg.Tags,
		PostingPolicy:          cfg.PostingPolicy,
		PrivacyPolicy:          cfg.PrivacyPolicy,
		PaymentsURL:            cfg.PaymentsURL,
		Retention:              cfg.Retention,
		NIP50:                  cfg.NIP50,
		SupportedNIPExtensions: cfg.SupportedNIPExtensions,
		SupportedGRASPs:        cfg.SupportedGRASPs,
	}

	if cfg.Limitation != nil && cfg.Limitation.HasValues() {
		doc.Limitation = cfg.Limitation
	}
	if cfg.Fees != nil && cfg.Fees.HasValues() {
		doc.Fees = cfg.Fees
	}

	return doc
}

// EffectiveIcon returns the explicitly configured icon, or the relay's
// embedded icon URL derived from its public address.
func (cfg *RelayInformationDocument) EffectiveIcon() string {
	if cfg == nil {
		return ""
	}
	if strings.TrimSpace(cfg.Icon) != "" {
		return cfg.Icon
	}

	if iconURL := iconURLFromBase(cfg.CanonicalURL, true); iconURL != "" {
		return iconURL
	}

	return iconURLFromBase(cfg.URL, false)
}

func iconURLFromBase(rawURL string, websocketURL bool) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return ""
	}

	switch parsed.Scheme {
	case "ws":
		if !websocketURL {
			return ""
		}
		parsed.Scheme = "http"
	case "wss":
		if !websocketURL {
			return ""
		}
		parsed.Scheme = "https"
	case "http", "https":
		if websocketURL {
			return ""
		}
	default:
		return ""
	}

	parsed.Path = "/nostr.png"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.User = nil
	return parsed.String()
}

func (cfg *RelayLimitationDocument) HasValues() bool {
	if cfg == nil {
		return false
	}
	return cfg.MaxMessageLength != nil ||
		cfg.MaxSubscriptions != nil ||
		cfg.MaxFilters != nil ||
		cfg.MaxLimit != nil ||
		cfg.DefaultLimit != nil ||
		cfg.MaxSubidLength != nil ||
		cfg.MaxEventTags != nil ||
		cfg.MaxContentLength != nil ||
		cfg.MinPowDifficulty != nil ||
		cfg.CreatedAtLowerLimit != nil ||
		cfg.CreatedAtUpperLimit != nil ||
		cfg.AuthRequired != nil ||
		cfg.PaymentRequired != nil ||
		cfg.RestrictedWrites != nil
}

func (cfg *RelayFeesDocument) HasValues() bool {
	if cfg == nil {
		return false
	}
	return len(cfg.Admission) > 0 || len(cfg.Subscription) > 0 || len(cfg.Publication) > 0
}

func (cfg *RelayInformationDocument) MaxSubscriptions() int {
	if cfg == nil || cfg.Limitation == nil || cfg.Limitation.MaxSubscriptions == nil {
		return 0
	}
	return *cfg.Limitation.MaxSubscriptions
}
