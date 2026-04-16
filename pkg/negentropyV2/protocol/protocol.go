package protocol

import (
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

const (
	VersionByte byte   = 0x61
	Infinity    uint64 = ^uint64(0)
)

type Mode uint64

const (
	ModeSkip        Mode = 0
	ModeFingerprint Mode = 1
	ModeIDList      Mode = 2
)

type Bound struct {
	Timestamp uint64
	Prefix    []byte
}

type Range struct {
	UpperBound  Bound
	Mode        Mode
	Fingerprint [16]byte
	IDs         []model.EventID
}

type Message struct {
	Version byte
	Ranges  []Range
}

var (
	ErrUnexpectedEOF  = errors.New("negentropy parse ends prematurely")
	ErrInvalidVersion = errors.New("invalid negentropy protocol version")
	ErrInvalidMode    = errors.New("unexpected negentropy mode")
)

func ParseMessage(raw []byte) (Message, error) {
	if len(raw) < 1 {
		return Message{}, ErrUnexpectedEOF
	}

	m := Message{Version: raw[0]}

	if m.Version < 0x60 || m.Version > 0x6F {
		return Message{}, fmt.Errorf("%w byte %x", ErrInvalidVersion, m.Version)
	}

	buf := newBuffer(raw[1:])
	lastTimestamp := uint64(0)

	for !buf.Empty() {
		bound, nextTimestamp, err := decodeBound(buf, lastTimestamp)
		if err != nil {
			return Message{}, err
		}
		lastTimestamp = nextTimestamp

		modeVal, err := decodeVarInt(buf)
		if err != nil {
			return Message{}, err
		}

		rng := Range{UpperBound: bound, Mode: Mode(modeVal)}

		switch rng.Mode {
		case ModeSkip:
		case ModeFingerprint:
			fp, err := buf.ReadN(16)
			if err != nil {
				return Message{}, err
			}
			copy(rng.Fingerprint[:], fp)
		case ModeIDList:
			length, err := decodeVarInt(buf)
			if err != nil {
				return Message{}, err
			}

			rng.IDs = make([]model.EventID, 0, length)
			for range length {
				bytes, err := buf.ReadN(model.IDSize)
				if err != nil {
					return Message{}, err
				}

				var id model.EventID
				copy(id[:], bytes)
				rng.IDs = append(rng.IDs, id)
			}
		default:
			return Message{}, fmt.Errorf("%w: %d", ErrInvalidMode, modeVal)
		}

		m.Ranges = append(m.Ranges, rng)
	}

	return m, nil
}

func EncodeMessage(msg Message) ([]byte, error) {
	if msg.Version < 0x60 || msg.Version > 0x6F {
		return nil, fmt.Errorf("%w byte %x", ErrInvalidVersion, msg.Version)
	}

	out := make([]byte, 0, 1024)
	out = append(out, msg.Version)

	lastTimestamp := uint64(0)

	for _, r := range msg.Ranges {
		bound, nextTimestamp := encodeBound(r.UpperBound, lastTimestamp)
		lastTimestamp = nextTimestamp
		out = append(out, bound...)

		out = append(out, encodeVarInt(uint64(r.Mode))...)

		switch r.Mode {
		case ModeSkip:
		case ModeFingerprint:
			out = append(out, r.Fingerprint[:]...)
		case ModeIDList:
			out = append(out, encodeVarInt(uint64(len(r.IDs)))...)
			for _, id := range r.IDs {
				out = append(out, id[:]...)
			}
		default:
			return nil, fmt.Errorf("%w: %d", ErrInvalidMode, r.Mode)
		}
	}

	return out, nil
}
