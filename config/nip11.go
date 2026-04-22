package config

type publicRelayInformationDocument struct {
	Name           string                   `json:"name,omitempty"`
	Description    string                   `json:"description,omitempty"`
	Banner         string                   `json:"banner,omitempty"`
	Icon           string                   `json:"icon,omitempty"`
	PubKey         string                   `json:"pubkey,omitempty"`
	Self           string                   `json:"self,omitempty"`
	Contact        string                   `json:"contact,omitempty"`
	SupportedNIPs  []int                    `json:"supported_nips,omitempty"`
	Software       string                   `json:"software,omitempty"`
	Version        string                   `json:"version,omitempty"`
	TermsOfService string                   `json:"terms_of_service,omitempty"`
	Limitation     *RelayLimitationDocument `json:"limitation,omitempty"`
	RelayCountries []string                 `json:"relay_countries,omitempty"`
	LanguageTags   []string                 `json:"language_tags,omitempty"`
	Tags           []string                 `json:"tags,omitempty"`
	PostingPolicy  string                   `json:"posting_policy,omitempty"`
	PaymentsURL    string                   `json:"payments_url,omitempty"`
	Fees           *RelayFeesDocument       `json:"fees,omitempty"`
}

func (cfg *RelayInformationDocument) PublicNIP11() any {
	if cfg == nil {
		return publicRelayInformationDocument{}
	}

	doc := publicRelayInformationDocument{
		Name:           cfg.Name,
		Description:    cfg.Description,
		Banner:         cfg.Banner,
		Icon:           cfg.Icon,
		PubKey:         cfg.PubKey,
		Self:           cfg.Self,
		Contact:        cfg.Contact,
		SupportedNIPs:  cfg.SupportedNIPs,
		Software:       cfg.Software,
		Version:        cfg.Version,
		TermsOfService: cfg.TermsOfService,
		RelayCountries: cfg.RelayCountries,
		LanguageTags:   cfg.LanguageTags,
		Tags:           cfg.Tags,
		PostingPolicy:  cfg.PostingPolicy,
		PaymentsURL:    cfg.PaymentsURL,
	}

	if cfg.Limitation != nil && cfg.Limitation.HasValues() {
		doc.Limitation = cfg.Limitation
	}
	if cfg.Fees != nil && cfg.Fees.HasValues() {
		doc.Fees = cfg.Fees
	}

	return doc
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
