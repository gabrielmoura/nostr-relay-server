package sync

type ConfSync struct {
	Remote     string // Remote Nostr Server URL
	Pk         string // Public Key to sync
	Direction  string // Direction of the sync (up, down, both)
	NumWorkers int    // Number of workers for parallel processing
	BatchSize  int    // Size of each batch for processing
}
