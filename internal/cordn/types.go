// Package cordn contains the Cordn coordinator domain hosted by NRServer.
// It intentionally has no dependency on RelaySync or the relay event store.
package cordn

import "time"

type KeyPackage struct {
	Reference       string
	StablePubkey    string
	Payload         []byte
	PublicationJSON []byte
	LastResort      bool
	PublishedAt     time.Time
}

type Welcome struct {
	TargetPubkey    string
	KeyPackageRef   string
	Payload         []byte
	JoinAfterCursor *int64
	CreatedAt       time.Time
}

type JoinRequest struct {
	GroupID         string
	RequesterPubkey string
	KeyPackageRef   string
	CreatedAt       time.Time
}

type GroupMessage struct {
	GroupID   string
	Cursor    int64
	Payload   []byte
	CreatedAt time.Time
}
