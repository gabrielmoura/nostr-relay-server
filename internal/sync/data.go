package sync

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type Direction string

const (
	DirectionBoth Direction = "both"
	DirectionDown Direction = "down"
	DirectionUp   Direction = "up"
	DirectionNone Direction = "none"
)

func ParseDirection(raw string) (Direction, error) {
	d := Direction(raw)
	switch d {
	case DirectionBoth, DirectionDown, DirectionUp, DirectionNone:
		return d, nil
	default:
		return "", fmt.Errorf("invalid direction %q: expected one of both/down/up/none", raw)
	}
}

func (d Direction) DoUp() bool {
	return d == DirectionBoth || d == DirectionUp
}

func (d Direction) DoDown() bool {
	return d == DirectionBoth || d == DirectionDown
}

// ConfSync define a configuração para uma operação de sincronização.
type ConfSync struct {
	Remote      string
	Pk          string
	Direction   Direction
	OpenFilter  any
	LocalFilter []nostr.Filter
	Timeout     time.Duration
}
