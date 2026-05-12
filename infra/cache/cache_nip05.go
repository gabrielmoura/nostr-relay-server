package cache

import (
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
)

const nip05DocKey = "nip05:doc"

func SetNIP05Doc(val string) error {
	return SetWithTTL(nip05DocKey, val, ttlOr(config.Cfg.Redis.Cache.NIP05DocTTL, 24*time.Hour))
}

func GetNIP05Doc() (string, bool) {
	val, err := Get(nip05DocKey)
	return val, err == nil
}

func DeleteNIP05Doc() error {
	return Delete(nip05DocKey)
}
