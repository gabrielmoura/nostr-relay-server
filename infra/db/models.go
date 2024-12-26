package db

import (
	"github.com/goccy/go-json"
	"time"
)

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

func (o *Object) ToJson() []byte {
	j, _ := json.Marshal(o)
	return j
}

type ObjectHash [32]byte

func StringToObjectHash(s string) ObjectHash {
	var h ObjectHash
	copy(h[:], s)
	return h
}

type Object struct {
	Hash            string    `json:"hash"`
	CreatedAt       time.Time `json:"created_at"`
	MimeType        string    `json:"mime_type"`
	Size            int64     `json:"size"`
	Blocked         bool      `json:"blocked"`
	ExpiresAt       time.Time `json:"expires_at"`
	BlockedByReason string    `json:"blocked_by_reason,omitempty"`
}

type ObjectResponse struct {
	Hash      string `json:"hash"`
	CreatedAt int64  `json:"created_at"`
	Url       string `json:"url"`
	MimeType  string `json:"mime_type"`
}
type ObjectResponseData struct {
	Hash      string `json:"hash"`
	CreatedAt int64  `json:"created_at"`
	Link      string `json:"link"`
	MimeType  string `json:"mime_type"`
}
