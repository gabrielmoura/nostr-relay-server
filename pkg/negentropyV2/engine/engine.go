package engine

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/protocol"
)

const (
	defaultBuckets           = 16
	defaultFrameSafetyMargin = 200
)

var ErrDuplicateItem = errors.New("duplicate item inserted")

type Options struct {
	FrameSizeLimit int
	Buckets        int
}

type Diff struct {
	Have []model.EventID
	Need []model.EventID
}

type Reconciler struct {
	items            []model.EventRef
	frameSizeLimit   int
	buckets          int
	frameSafetyLimit int
}

func NewReconciler(refs []model.EventRef, opts Options) (*Reconciler, error) {
	items := make([]model.EventRef, len(refs))
	copy(items, refs)

	slices.SortFunc(items, compareEventRef)

	for i := 1; i < len(items); i++ {
		if compareEventRef(items[i-1], items[i]) == 0 {
			return nil, ErrDuplicateItem
		}
	}

	buckets := opts.Buckets
	if buckets <= 0 {
		buckets = defaultBuckets
	}

	rec := &Reconciler{
		items:            items,
		frameSizeLimit:   opts.FrameSizeLimit,
		buckets:          buckets,
		frameSafetyLimit: defaultFrameSafetyMargin,
	}

	return rec, nil
}

func (r *Reconciler) Initiate() ([]byte, error) {
	msg := protocol.Message{Version: protocol.VersionByte}
	ranges := r.splitRange(0, len(r.items), protocol.Bound{Timestamp: protocol.Infinity})
	msg.Ranges = append(msg.Ranges, ranges...)

	out, err := protocol.EncodeMessage(msg)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (r *Reconciler) ReconcileAsResponder(query []byte) ([]byte, error) {
	parsed, err := protocol.ParseMessage(query)
	if err != nil {
		return nil, err
	}

	if parsed.Version != protocol.VersionByte {
		return []byte{protocol.VersionByte}, nil
	}

	response := protocol.Message{Version: protocol.VersionByte}
	prevBound := protocol.Bound{}
	prevIndex := 0
	skipPending := false

	for _, incoming := range parsed.Ranges {
		lower := prevIndex
		upper := r.findLowerBound(prevIndex, len(r.items), incoming.UpperBound)

		appendSkip := func() {
			if !skipPending {
				return
			}

			skipPending = false
			response.Ranges = append(response.Ranges, protocol.Range{UpperBound: prevBound, Mode: protocol.ModeSkip})
		}

		switch incoming.Mode {
		case protocol.ModeSkip:
			skipPending = true
		case protocol.ModeFingerprint:
			ours := r.fingerprint(lower, upper)
			if ours == incoming.Fingerprint {
				skipPending = true
			} else {
				appendSkip()
				response.Ranges = append(response.Ranges, r.splitRange(lower, upper, incoming.UpperBound)...)
			}
		case protocol.ModeIDList:
			appendSkip()

			bound := incoming.UpperBound
			ids := make([]model.EventID, 0, upper-lower)

			for idx := lower; idx < upper; idx++ {
				if r.exceededFrameSizeLimitEstimate(response, len(ids)*model.IDSize) {
					if idx > lower {
						bound = minimalBound(r.items[idx-1], r.items[idx])
						upper = idx
					}
					break
				}

				ids = append(ids, r.items[idx].ID)
			}

			response.Ranges = append(response.Ranges, protocol.Range{UpperBound: bound, Mode: protocol.ModeIDList, IDs: ids})
		default:
			return nil, fmt.Errorf("unexpected mode: %d", incoming.Mode)
		}

		if r.exceededFrameSizeLimit(response) {
			response.Ranges = append(response.Ranges, protocol.Range{
				UpperBound:  protocol.Bound{Timestamp: protocol.Infinity},
				Mode:        protocol.ModeFingerprint,
				Fingerprint: r.fingerprint(upper, len(r.items)),
			})
			break
		}

		prevBound = incoming.UpperBound
		prevIndex = upper
	}

	out, err := protocol.EncodeMessage(response)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (r *Reconciler) ReconcileAsInitiator(query []byte) ([]byte, Diff, bool, error) {
	parsed, err := protocol.ParseMessage(query)
	if err != nil {
		return nil, Diff{}, false, err
	}

	if parsed.Version != protocol.VersionByte {
		return nil, Diff{}, false, fmt.Errorf("server requested unsupported protocol version: %x", parsed.Version)
	}

	diff := Diff{}
	response := protocol.Message{Version: protocol.VersionByte}
	prevBound := protocol.Bound{}
	prevIndex := 0
	skipPending := false

	for _, incoming := range parsed.Ranges {
		lower := prevIndex
		upper := r.findLowerBound(prevIndex, len(r.items), incoming.UpperBound)

		appendSkip := func() {
			if !skipPending {
				return
			}

			skipPending = false
			response.Ranges = append(response.Ranges, protocol.Range{UpperBound: prevBound, Mode: protocol.ModeSkip})
		}

		switch incoming.Mode {
		case protocol.ModeSkip:
			skipPending = true
		case protocol.ModeFingerprint:
			ours := r.fingerprint(lower, upper)
			if ours == incoming.Fingerprint {
				skipPending = true
			} else {
				appendSkip()
				response.Ranges = append(response.Ranges, r.splitRange(lower, upper, incoming.UpperBound)...)
			}
		case protocol.ModeIDList:
			skipPending = true

			their := make(map[model.EventID]struct{}, len(incoming.IDs))
			for _, id := range incoming.IDs {
				their[id] = struct{}{}
			}

			for idx := lower; idx < upper; idx++ {
				id := r.items[idx].ID
				if _, ok := their[id]; ok {
					delete(their, id)
					continue
				}

				diff.Have = append(diff.Have, id)
			}

			for id := range their {
				diff.Need = append(diff.Need, id)
			}
		default:
			return nil, Diff{}, false, fmt.Errorf("unexpected mode: %d", incoming.Mode)
		}

		if r.exceededFrameSizeLimit(response) {
			response.Ranges = append(response.Ranges, protocol.Range{
				UpperBound:  protocol.Bound{Timestamp: protocol.Infinity},
				Mode:        protocol.ModeFingerprint,
				Fingerprint: r.fingerprint(upper, len(r.items)),
			})
			break
		}

		prevBound = incoming.UpperBound
		prevIndex = upper
	}

	out, err := protocol.EncodeMessage(response)
	if err != nil {
		return nil, Diff{}, false, err
	}

	done := len(out) == 1
	if done {
		return nil, diff, true, nil
	}

	return out, diff, false, nil
}

func (r *Reconciler) splitRange(lower int, upper int, upperBound protocol.Bound) []protocol.Range {
	n := upper - lower
	if n <= 0 {
		return []protocol.Range{{UpperBound: upperBound, Mode: protocol.ModeIDList, IDs: nil}}
	}

	if n < r.buckets*2 {
		ids := make([]model.EventID, 0, n)
		for i := lower; i < upper; i++ {
			ids = append(ids, r.items[i].ID)
		}

		return []protocol.Range{{UpperBound: upperBound, Mode: protocol.ModeIDList, IDs: ids}}
	}

	itemsPerBucket := n / r.buckets
	bucketsWithExtra := n % r.buckets
	current := lower
	out := make([]protocol.Range, 0, r.buckets)

	for i := 0; i < r.buckets; i++ {
		bucketSize := itemsPerBucket
		if i < bucketsWithExtra {
			bucketSize++
		}

		next := current + bucketSize
		bound := upperBound

		if next < upper {
			bound = minimalBound(r.items[next-1], r.items[next])
		}

		out = append(out, protocol.Range{
			UpperBound:  bound,
			Mode:        protocol.ModeFingerprint,
			Fingerprint: r.fingerprint(current, next),
		})

		current = next
	}

	return out
}

func (r *Reconciler) findLowerBound(begin int, end int, bound protocol.Bound) int {
	targetID := model.EventID{}
	copy(targetID[:], bound.Prefix)

	target := model.EventRef{CreatedAt: bound.Timestamp, ID: targetID}

	index := slices.IndexFunc(r.items[begin:end], func(item model.EventRef) bool {
		return compareEventRef(item, target) >= 0
	})

	if index < 0 {
		return end
	}

	return begin + index
}

func (r *Reconciler) fingerprint(begin int, end int) [16]byte {
	sum := [32]byte{}

	for i := begin; i < end; i++ {
		sum = addLE(sum, r.items[i].ID)
	}

	payload := make([]byte, 0, 64)
	payload = append(payload, sum[:]...)
	payload = append(payload, encodeVarInt(uint64(end-begin))...)

	hash := sha256.Sum256(payload)
	out := [16]byte{}
	copy(out[:], hash[:16])

	return out
}

func (r *Reconciler) exceededFrameSizeLimit(msg protocol.Message) bool {
	if r.frameSizeLimit == 0 {
		return false
	}

	encoded, err := protocol.EncodeMessage(msg)
	if err != nil {
		return false
	}

	return len(encoded) > r.frameSizeLimit-r.frameSafetyLimit
}

func (r *Reconciler) exceededFrameSizeLimitEstimate(msg protocol.Message, idBytes int) bool {
	if r.frameSizeLimit == 0 {
		return false
	}

	encoded, err := protocol.EncodeMessage(msg)
	if err != nil {
		return false
	}

	return len(encoded)+idBytes > r.frameSizeLimit-r.frameSafetyLimit
}

func minimalBound(prev model.EventRef, curr model.EventRef) protocol.Bound {
	if curr.CreatedAt != prev.CreatedAt {
		return protocol.Bound{Timestamp: curr.CreatedAt}
	}

	shared := 0
	for i := 0; i < model.IDSize; i++ {
		if prev.ID[i] != curr.ID[i] {
			break
		}

		shared++
	}

	prefix := make([]byte, shared+1)
	copy(prefix, curr.ID[:shared+1])

	return protocol.Bound{Timestamp: curr.CreatedAt, Prefix: prefix}
}

func compareEventRef(a model.EventRef, b model.EventRef) int {
	if a.CreatedAt < b.CreatedAt {
		return -1
	}

	if a.CreatedAt > b.CreatedAt {
		return 1
	}

	return bytes.Compare(a.ID[:], b.ID[:])
}

func addLE(acc [32]byte, id model.EventID) [32]byte {
	carry := uint16(0)

	for i := 0; i < 32; i++ {
		total := uint16(acc[i]) + uint16(id[i]) + carry
		acc[i] = byte(total)
		carry = total >> 8
	}

	return acc
}

func encodeVarInt(value uint64) []byte {
	if value == 0 {
		return []byte{0}
	}

	buf := make([]byte, 0, 10)
	for value > 0 {
		buf = append(buf, byte(value&0x7F))
		value >>= 7
	}

	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}

	for i := 0; i < len(buf)-1; i++ {
		buf[i] |= 0x80
	}

	return buf
}
