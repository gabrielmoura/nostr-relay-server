package db

import (
	"github.com/goccy/go-json"
	"time"
)

type BannedUser struct {
	User   Profile `json:"user"`
	Reason string  `json:"reason"`
	ID     int64   `json:"id"`
}

type Profile struct {
	PublicKey   string `json:"public_key"`
	Name        string `json:"name"`
	About       string `json:"about,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Banner      string `json:"banner,omitempty"`
	Website     string `json:"website,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Lud16       string `json:"lud16,omitempty"`
	Pronouns    string `json:"pronouns,omitempty"`
	Nip05       string `json:"nip05,omitempty"`
	ID          int64  `json:"id"`
	Bot         bool   `json:"bot,omitempty"`
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
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Hash            string    `json:"hash"`
	MimeType        string    `json:"mime_type"`
	BlockedByReason string    `json:"blocked_by_reason,omitempty"`
	Size            int64     `json:"size"`
	Blocked         bool      `json:"blocked"`
	PublicKey       string    `json:"public_key"`
	Tags            []byte    `json:"tags,omitempty"`
}

type ObjectResponse struct {
	Hash      string `json:"hash"`
	Url       string `json:"url"`
	MimeType  string `json:"mime_type"`
	CreatedAt int64  `json:"created_at"`
}
type ObjectResponseData struct {
	Hash      string `json:"hash"`
	Link      string `json:"link"`
	MimeType  string `json:"mime_type"`
	CreatedAt int64  `json:"created_at"`
}
