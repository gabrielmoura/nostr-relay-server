package nip86

import (
	"context"
	"encoding/json"

	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

const (
	MethodSupportedMethods   = "supportedmethods"
	MethodBanPubKey          = "banpubkey"
	MethodUnbanPubKey        = "unbanpubkey"
	MethodListBannedPubKeys  = "listbannedpubkeys"
	MethodAllowPubKey        = "allowpubkey"
	MethodUnallowPubKey      = "unallowpubkey"
	MethodListAllowedPubKeys = "listallowedpubkeys"
	MethodAllowEvent         = "allowevent"
	MethodBanEvent           = "banevent"
	MethodListBannedEvents   = "listbannedevents"
	MethodChangeRelayName    = "changerelayname"
	MethodChangeRelayDesc    = "changerelaydescription"
	MethodBlockIP            = "blockip"
	MethodUnblockIP          = "unblockip"
	MethodListBlockedIPs     = "listblockedips"
	contentTypeJSONRPC       = "application/nostr+json+rpc"
	cacheStatePresent        = "1"
	cacheStateMissing        = "0"
)

var supportedMethods = []string{
	MethodSupportedMethods,
	MethodBanPubKey,
	MethodUnbanPubKey,
	MethodListBannedPubKeys,
	MethodAllowPubKey,
	MethodUnallowPubKey,
	MethodListAllowedPubKeys,
	MethodAllowEvent,
	MethodBanEvent,
	MethodListBannedEvents,
	MethodChangeRelayName,
	MethodChangeRelayDesc,
	MethodBlockIP,
	MethodUnblockIP,
	MethodListBlockedIPs,
}

type Request struct {
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

type Response struct {
	HTTPStatus int    `json:"-"`
	Result     any    `json:"result"`
	Error      string `json:"error,omitempty"`
}

type CallContext struct {
	AdminPubKey string
	RemoteIP    string
}

type Repository interface {
	BanUserByPubKey(ctx context.Context, key, reason string, relatedIDs []string) error
	UnbanUserByPubKey(ctx context.Context, key string) error
	ListBannedPubKeys(ctx context.Context) ([]db.NIP86PubKeyRecord, error)
	UpsertAllowedPubKey(ctx context.Context, pubkey, reason, createdBy string) error
	DeleteAllowedPubKey(ctx context.Context, pubkey string) error
	ListAllowedPubKeys(ctx context.Context) ([]db.NIP86PubKeyRecord, error)
	UpsertBannedEvent(ctx context.Context, eventID, reason, createdBy string) error
	DeleteBannedEvent(ctx context.Context, eventID string) error
	GetBannedEvent(ctx context.Context, eventID string) (db.NIP86EventRecord, bool, error)
	ListBannedEvents(ctx context.Context) ([]db.NIP86EventRecord, error)
	DeleteEvent(ctx context.Context, id, reasonID string) error
	UpsertBlockedIP(ctx context.Context, ip, reason, createdBy string) error
	DeleteBlockedIP(ctx context.Context, ip string) error
	GetBlockedIP(ctx context.Context, ip string) (db.NIP86IPRecord, bool, error)
	ListBlockedIPs(ctx context.Context) ([]db.NIP86IPRecord, error)
	UpsertRelayMetadata(ctx context.Context, relayURL, name, description, updatedBy string) error
	GetRelayMetadata(ctx context.Context, relayURL string) (db.NIP86RelayMetadataRecord, bool, error)
}

type cacheEntry struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type AuthInput struct {
	Authorization string
	Method        string
	URL           string
	Body          []byte
}

type AuthResult struct {
	PubKey string
	Event  nostr.Event
}
