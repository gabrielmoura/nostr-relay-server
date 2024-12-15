package db

type BannedUser struct {
	ID     int64   `json:"id"`
	User   Profile `json:"user"`
	Reason string  `json:"reason"` // Reason for ban
}

type Profile struct {
	ID          int64  `json:"id"`
	PublicKey   string `json:"public_key"`
	Name        string `json:"name"`
	About       string `json:"about,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Bot         bool   `json:"bot,omitempty"`
	Banner      string `json:"banner,omitempty"`
	Website     string `json:"website,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Lud16       string `json:"lud16,omitempty"`
	Pronouns    string `json:"pronouns,omitempty"`
	Nip05       string `json:"nip05,omitempty"`
}
