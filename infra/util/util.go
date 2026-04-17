package util

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/tmthrgd/go-hex"
	"go.uber.org/zap"
)

func AuthTimeCheck(eventCreatedAt int64) (bool, string) {
	currentTime := time.Now()
	eventTime := time.Unix(eventCreatedAt, 0)
	tenMinutesAgo := currentTime.Add(-10 * time.Minute)

	// Check if the event timestamp is within the last 10 minutes
	if eventTime.Before(tenMinutesAgo) {
		errMsg := fmt.Sprintf("invalid: event creation date is more than 10 minutes ago (%s)", eventTime)
		return false, errMsg
	}

	return true, ""
}

func GenChallenge() string {
	// NIP-42 challenge
	challenge := make([]byte, 32)
	_, err := rand.Read(challenge)
	if err != nil {
		log.Logger.Warn("error generating challenge", zap.Error(err))
		return ""
	}
	return hex.EncodeToString(challenge)
}
