package relaysync

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

const (
	NegentropyVersion  byte   = 0x61
	negentropyInfinity uint64 = ^uint64(0)
)

type NegentropyMode uint8

const (
	NegentropySkip NegentropyMode = iota
	NegentropyFingerprint
	NegentropyIDs
)

type NegentropyBound struct {
	Timestamp uint64
	Prefix    []byte
}

type NegentropyRange struct {
	UpperBound  NegentropyBound
	Mode        NegentropyMode
	Fingerprint [16]byte
	IDs         []EventID
}

type NegentropyMessage struct {
	Version byte
	Ranges  []NegentropyRange
}

var (
	ErrNegentropyEOF     = errors.New("negentropy frame truncated")
	ErrNegentropyVersion = errors.New("unsupported negentropy version")
	ErrNegentropyMode    = errors.New("invalid negentropy mode")
	ErrNegentropyBound   = errors.New("invalid negentropy bound")
)

func EncodeNegentropy(message NegentropyMessage) ([]byte, error) {
	if message.Version < 0x60 || message.Version > 0x6f {
		return nil, fmt.Errorf("%w: 0x%x", ErrNegentropyVersion, message.Version)
	}
	out := []byte{message.Version}
	var last uint64
	for _, item := range message.Ranges {
		bound, next, err := encodeNegentropyBound(item.UpperBound, last)
		if err != nil {
			return nil, err
		}
		out = append(out, bound...)
		last = next
		out = append(out, encodeNegentropyVarint(uint64(item.Mode))...)
		switch item.Mode {
		case NegentropySkip:
		case NegentropyFingerprint:
			out = append(out, item.Fingerprint[:]...)
		case NegentropyIDs:
			out = append(out, encodeNegentropyVarint(uint64(len(item.IDs)))...)
			for _, id := range item.IDs {
				out = append(out, id[:]...)
			}
		default:
			return nil, fmt.Errorf("%w: %d", ErrNegentropyMode, item.Mode)
		}
	}
	return out, nil
}

func DecodeNegentropy(raw []byte) (NegentropyMessage, error) {
	if len(raw) == 0 {
		return NegentropyMessage{}, ErrNegentropyEOF
	}
	if raw[0] < 0x60 || raw[0] > 0x6f {
		return NegentropyMessage{}, ErrNegentropyVersion
	}
	reader := negentropyReader{data: raw[1:]}
	message := NegentropyMessage{Version: raw[0]}
	var last uint64
	for reader.remaining() > 0 {
		bound, next, err := decodeNegentropyBound(&reader, last)
		if err != nil {
			return NegentropyMessage{}, err
		}
		last = next
		mode, err := reader.varint()
		if err != nil {
			return NegentropyMessage{}, err
		}
		item := NegentropyRange{UpperBound: bound, Mode: NegentropyMode(mode)}
		switch item.Mode {
		case NegentropySkip:
		case NegentropyFingerprint:
			value, err := reader.read(16)
			if err != nil {
				return NegentropyMessage{}, err
			}
			copy(item.Fingerprint[:], value)
		case NegentropyIDs:
			count, err := reader.varint()
			if err != nil || count > uint64(reader.remaining()/32) {
				return NegentropyMessage{}, ErrNegentropyEOF
			}
			item.IDs = make([]EventID, count)
			for i := range item.IDs {
				value, err := reader.read(32)
				if err != nil {
					return NegentropyMessage{}, err
				}
				copy(item.IDs[i][:], value)
			}
		default:
			return NegentropyMessage{}, fmt.Errorf("%w: %d", ErrNegentropyMode, mode)
		}
		message.Ranges = append(message.Ranges, item)
	}
	return message, nil
}

type NegentropyDiff struct {
	Have []EventID
	Need []EventID
}

type NegentropyOptions struct {
	FrameSizeLimit int
	Buckets        int
}

type Negentropy struct {
	items               []EventMetadata
	frameLimit, buckets int
}

func NewNegentropy(items []EventMetadata, options NegentropyOptions) (*Negentropy, error) {
	vector, err := NewVector(items)
	if err != nil {
		return nil, err
	}
	buckets := options.Buckets
	if buckets <= 0 {
		buckets = 16
	}
	return &Negentropy{items: append([]EventMetadata(nil), vector.items...), frameLimit: options.FrameSizeLimit, buckets: buckets}, nil
}

func (n *Negentropy) Initiate() ([]byte, error) {
	frame, err := n.encodeRanges(n.split(0, len(n.items), NegentropyBound{Timestamp: negentropyInfinity}))
	if err != nil {
		return nil, err
	}
	return n.limitFrame(frame)
}

func (n *Negentropy) Respond(frame []byte) ([]byte, error) {
	message, err := DecodeNegentropy(frame)
	if err != nil {
		return nil, err
	}
	return n.respond(message)
}

func (n *Negentropy) Apply(frame []byte) ([]byte, NegentropyDiff, bool, error) {
	message, err := DecodeNegentropy(frame)
	if err != nil {
		return nil, NegentropyDiff{}, false, err
	}
	response := NegentropyMessage{Version: NegentropyVersion}
	var diff NegentropyDiff
	var previous int
	var previousBound NegentropyBound
	for _, incoming := range message.Ranges {
		upper := n.lowerBound(previous, incoming.UpperBound)
		switch incoming.Mode {
		case NegentropySkip:
		case NegentropyFingerprint:
			if n.fingerprint(previous, upper) != incoming.Fingerprint {
				response.Ranges = append(response.Ranges, n.split(previous, upper, incoming.UpperBound)...)
			}
		case NegentropyIDs:
			remote := make(map[EventID]struct{}, len(incoming.IDs))
			for _, id := range incoming.IDs {
				remote[id] = struct{}{}
			}
			for _, item := range n.items[previous:upper] {
				if _, ok := remote[item.ID]; ok {
					delete(remote, item.ID)
				} else {
					diff.Have = append(diff.Have, item.ID)
				}
			}
			for id := range remote {
				diff.Need = append(diff.Need, id)
			}
		default:
			return nil, NegentropyDiff{}, false, ErrNegentropyMode
		}
		previous, previousBound = upper, incoming.UpperBound
		_ = previousBound
	}
	encoded, err := EncodeNegentropy(response)
	if err != nil {
		return nil, NegentropyDiff{}, false, err
	}
	if encoded, err = n.limitFrame(encoded); err != nil {
		return nil, NegentropyDiff{}, false, err
	}
	return encoded, diff, len(encoded) == 1, nil
}

func (n *Negentropy) respond(message NegentropyMessage) ([]byte, error) {
	response := NegentropyMessage{Version: NegentropyVersion}
	previous := 0
	for _, incoming := range message.Ranges {
		upper := n.lowerBound(previous, incoming.UpperBound)
		switch incoming.Mode {
		case NegentropySkip:
		case NegentropyFingerprint:
			if n.fingerprint(previous, upper) != incoming.Fingerprint {
				response.Ranges = append(response.Ranges, n.split(previous, upper, incoming.UpperBound)...)
			}
		case NegentropyIDs:
			response.Ranges = append(response.Ranges, NegentropyRange{UpperBound: incoming.UpperBound, Mode: NegentropyIDs, IDs: n.ids(previous, upper)})
		default:
			return nil, ErrNegentropyMode
		}
		previous = upper
	}
	encoded, err := EncodeNegentropy(response)
	if err != nil {
		return nil, err
	}
	return n.limitFrame(encoded)
}

func (n *Negentropy) limitFrame(frame []byte) ([]byte, error) {
	if n.frameLimit > 0 && len(frame) > n.frameLimit {
		return nil, ErrFrameTooLarge
	}
	return frame, nil
}

func (n *Negentropy) split(start, end int, bound NegentropyBound) []NegentropyRange {
	if end-start <= n.buckets*2 {
		return []NegentropyRange{{UpperBound: bound, Mode: NegentropyIDs, IDs: n.ids(start, end)}}
	}
	result := make([]NegentropyRange, 0, n.buckets)
	count := end - start
	for i, cursor := 0, start; i < n.buckets; i++ {
		next := start + (count*(i+1))/n.buckets
		if i == n.buckets-1 {
			next = end
		}
		upper := bound
		if next < end {
			upper = minimalNegentropyBound(n.items[next-1], n.items[next])
		}
		result = append(result, NegentropyRange{UpperBound: upper, Mode: NegentropyFingerprint, Fingerprint: n.fingerprint(cursor, next)})
		cursor = next
	}
	return result
}

func (n *Negentropy) ids(start, end int) []EventID {
	ids := make([]EventID, end-start)
	for i := range ids {
		ids[i] = n.items[start+i].ID
	}
	return ids
}
func (n *Negentropy) lowerBound(start int, bound NegentropyBound) int {
	target := EventMetadata{CreatedAt: int64(bound.Timestamp)}
	if bound.Timestamp == negentropyInfinity {
		return len(n.items)
	}
	copy(target.ID[:], bound.Prefix)
	for start < len(n.items) && compareMetadata(n.items[start], target) < 0 {
		start++
	}
	return start
}
func (n *Negentropy) fingerprint(start, end int) [16]byte {
	sum := [32]byte{}
	for _, item := range n.items[start:end] {
		carry := uint16(0)
		for i := range sum {
			total := uint16(sum[i]) + uint16(item.ID[i]) + carry
			sum[i] = byte(total)
			carry = total >> 8
		}
	}
	payload := append(sum[:], encodeNegentropyVarint(uint64(end-start))...)
	digest := sha256.Sum256(payload)
	var out [16]byte
	copy(out[:], digest[:16])
	return out
}
func (n *Negentropy) encodeRanges(ranges []NegentropyRange) ([]byte, error) {
	return EncodeNegentropy(NegentropyMessage{Version: NegentropyVersion, Ranges: ranges})
}

func minimalNegentropyBound(previous, current EventMetadata) NegentropyBound {
	bound := NegentropyBound{Timestamp: uint64(current.CreatedAt)}
	if current.CreatedAt != previous.CreatedAt {
		return bound
	}
	shared := 0
	for shared < len(current.ID) && current.ID[shared] == previous.ID[shared] {
		shared++
	}
	if shared < len(current.ID) {
		shared++
	}
	bound.Prefix = append([]byte(nil), current.ID[:shared]...)
	return bound
}

type negentropyReader struct {
	data []byte
	pos  int
}

func (r *negentropyReader) remaining() int { return len(r.data) - r.pos }
func (r *negentropyReader) read(size int) ([]byte, error) {
	if size < 0 || r.remaining() < size {
		return nil, ErrNegentropyEOF
	}
	value := r.data[r.pos : r.pos+size]
	r.pos += size
	return value, nil
}
func (r *negentropyReader) varint() (uint64, error) {
	var value uint64
	for i := 0; i < 10; i++ {
		b, err := r.read(1)
		if err != nil {
			return 0, err
		}
		value = (value << 7) | uint64(b[0]&0x7f)
		if b[0]&0x80 == 0 {
			return value, nil
		}
	}
	return 0, ErrNegentropyBound
}
func encodeNegentropyVarint(value uint64) []byte {
	if value == 0 {
		return []byte{0}
	}
	var reversed [10]byte
	i := len(reversed)
	for value > 0 {
		i--
		reversed[i] = byte(value & 0x7f)
		value >>= 7
	}
	out := append([]byte(nil), reversed[i:]...)
	for j := 0; j < len(out)-1; j++ {
		out[j] |= 0x80
	}
	return out
}
func encodeNegentropyBound(bound NegentropyBound, last uint64) ([]byte, uint64, error) {
	if len(bound.Prefix) > 32 {
		return nil, 0, ErrNegentropyBound
	}
	if bound.Timestamp == negentropyInfinity {
		return append(encodeNegentropyVarint(0), encodeNegentropyVarint(uint64(len(bound.Prefix)))...), negentropyInfinity, nil
	}
	if bound.Timestamp < last && last != negentropyInfinity {
		return nil, 0, ErrNegentropyBound
	}
	delta := bound.Timestamp
	if last != negentropyInfinity {
		delta -= last
	}
	out := append(encodeNegentropyVarint(delta+1), encodeNegentropyVarint(uint64(len(bound.Prefix)))...)
	return append(out, bound.Prefix...), bound.Timestamp, nil
}
func decodeNegentropyBound(reader *negentropyReader, last uint64) (NegentropyBound, uint64, error) {
	encoded, err := reader.varint()
	if err != nil {
		return NegentropyBound{}, 0, err
	}
	prefixLen, err := reader.varint()
	if err != nil || prefixLen > 32 {
		return NegentropyBound{}, 0, ErrNegentropyBound
	}
	prefix, err := reader.read(int(prefixLen))
	if err != nil {
		return NegentropyBound{}, 0, err
	}
	if encoded == 0 {
		return NegentropyBound{Timestamp: negentropyInfinity, Prefix: append([]byte(nil), prefix...)}, negentropyInfinity, nil
	}
	timestamp := encoded - 1
	if last != negentropyInfinity {
		timestamp += last
	}
	return NegentropyBound{Timestamp: timestamp, Prefix: append([]byte(nil), prefix...)}, timestamp, nil
}
