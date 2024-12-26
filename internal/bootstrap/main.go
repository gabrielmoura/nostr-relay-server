package bootstrap

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	nostrcustom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"go.uber.org/zap"
)

func CreateInitialEvents() {
	privKey := nostr.GeneratePrivateKey()
	pubKey, _ := nostr.GetPublicKey(privKey)

	npub, _ := nip19.EncodePublicKey(pubKey)
	npriv, _ := nip19.EncodePrivateKey(privKey)

	var events []nostr.Event

	// Create a profile metadata event
	pmeta, _ := json.Marshal(ProfileMetadata{
		Name:        config.Cfg.RelayInformation.Name,
		DisplayName: "Bot " + config.Cfg.RelayInformation.Name,
		About:       "I am a bot Relay",
		Website:     config.Cfg.RelayInformation.URL,
		Picture:     config.Cfg.RelayInformation.Icon,
		Banner:      config.Cfg.RelayInformation.Icon,
		Bot:         true,
	})

	pev := nostr.Event{
		PubKey:    pubKey,
		Content:   string(pmeta),
		Kind:      nostr.KindProfileMetadata,
		CreatedAt: nostr.Now(),
	}
	pev.Sign(privKey)
	events = append(events, pev)

	// Create a relay Info Event
	rdata, _ := json.Marshal(RelayInfo{
		Name:          config.Cfg.RelayInformation.Name,
		Description:   config.Cfg.RelayInformation.Description,
		PubKey:        npub,
		Contact:       config.Cfg.RelayInformation.Contact,
		SupportedNips: config.Cfg.RelayInformation.SupportedNIPs,
		Software:      config.Cfg.RelayInformation.Software,
		Version:       config.Cfg.RelayInformation.Version,
	})

	rev := nostr.Event{
		PubKey:    pubKey,
		Kind:      nostrcustom.KindRelay,
		CreatedAt: nostr.Now(),
		Content:   string(rdata),
	}
	rev.Sign(privKey)
	events = append(events, rev)

	// Create a Relay List Event
	rlev := nostr.Event{
		PubKey:    pubKey,
		Kind:      nostr.KindRelayListMetadata,
		CreatedAt: nostr.Now(),
		Content:   "",
		Tags: nostr.Tags{
			nostr.Tag{"r", config.Cfg.RelayInformation.CanonicalURL},
		},
	}
	rlev.Sign(privKey)
	events = append(events, rlev)

	// Create a Blossom Server list Event
	//if config.Cfg.Store.Enabled {
	//KindFileStorageServerList
	rsev := nostr.Event{
		PubKey:    pubKey,
		Kind:      nostr.KindUserServerList,
		CreatedAt: nostr.Now(),
		Content:   "",
		Tags: nostr.Tags{
			nostr.Tag{"server", config.Cfg.RelayInformation.URL},
		},
	}
	rsev.Sign(privKey)
	events = append(events, rsev)
	//}

	for _, ev := range events {
		db.DbQueries.InsertEvent(context.TODO(), &ev)
	}
	config.Cfg.RelayInformation.PrivKey = privKey
	config.Cfg.RelayInformation.PubKey = pubKey

	log.Logger.Info("Initial events created, save private and public keys and put in settings",
		zap.String("pub_key", pubKey),
		zap.String("priv_key", privKey),
		zap.String("npub", npub),
		zap.String("npriv", npriv),
	)

}

type RelayInfo struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PubKey        string `json:"pubkey"`
	Contact       string `json:"contact"`
	SupportedNips []int  `json:"supported_nips"`
	Software      string `json:"software"`
	Version       string `json:"version"`
}

type ProfileMetadata struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	About       string `json:"about,omitempty"`
	Website     string `json:"website,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Banner      string `json:"banner,omitempty"`
	NIP05       string `json:"nip05,omitempty"`
	LUD16       string `json:"lud16,omitempty"`

	Bot bool `json:"bot,omitempty"`
}
