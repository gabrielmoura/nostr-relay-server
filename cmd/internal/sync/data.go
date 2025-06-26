package sync

type ConfSync struct {
	Remote    string // Remote Nostr Server URL
	Pk        string // Public Key to sync
	Direction string // Direction of the sync (up, down, both)
}
