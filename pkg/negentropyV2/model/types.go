package model

import (
	"encoding/hex"
	"errors"
	"fmt"
)

const IDSize = 32

var ErrInvalidIDHex = errors.New("invalid event id hex")

type EventID [IDSize]byte

func ParseEventIDHex(value string) (EventID, error) {
	var id EventID

	if len(value) != IDSize*2 {
		return id, fmt.Errorf("%w: expected %d hex chars", ErrInvalidIDHex, IDSize*2)
	}

	buf, err := hex.DecodeString(value)
	if err != nil {
		return id, fmt.Errorf("%w: %v", ErrInvalidIDHex, err)
	}

	copy(id[:], buf)

	return id, nil
}

func (id EventID) Hex() string {
	return hex.EncodeToString(id[:])
}

type EventRef struct {
	CreatedAt uint64
	ID        EventID
}

type Filter struct {
	IDs     []string
	Authors []string
	Kinds   []int
	Tags    map[string][]string
	Search  string
	Since   *uint64
	Until   *uint64
	Limit   *int
}

type OpenRequest struct {
	SessionID         string
	Filter            Filter
	InitialMessageHex string
}

type MessageRequest struct {
	SessionID  string
	MessageHex string
}

type ResponseType string

const (
	ResponseTypeMessage ResponseType = "NEG-MSG"
	ResponseTypeError   ResponseType = "NEG-ERR"
	ResponseTypeClosed  ResponseType = "NEG-CLOSE"
)

type Response struct {
	Type       ResponseType
	SessionID  string
	MessageHex string
	Reason     string
	Done       bool
}
