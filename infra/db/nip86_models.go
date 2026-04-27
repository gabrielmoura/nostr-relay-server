package db

import "time"

type NIP86PubKeyRecord struct {
	PubKey    string    `json:"pubkey"`
	Reason    string    `json:"reason,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type NIP86EventRecord struct {
	EventID   string    `json:"event_id"`
	Reason    string    `json:"reason,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type NIP86IPRecord struct {
	IP        string    `json:"ip"`
	Reason    string    `json:"reason,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type NIP86RelayMetadataRecord struct {
	RelayURL    string    `json:"relay_url"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}
