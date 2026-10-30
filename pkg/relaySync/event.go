package relaysync

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

const (
	eventIDSize   = 32
	publicKeySize = 32
	signatureSize = 64
)

// EventID is a SHA-256 NIP-01 event identifier.
type EventID [eventIDSize]byte

// PublicKey is an x-only BIP-340 public key.
type PublicKey [publicKeySize]byte

// Signature is a BIP-340 Schnorr signature.
type Signature [signatureSize]byte

// Tag retains NIP-01 tag order and duplicate values.
type Tag []string

// Event is RelaySync's NIP-01 event representation. It intentionally has no
// dependency on a host relay's event type.
type Event struct {
	ID        EventID
	PublicKey PublicKey
	Signature Signature
	CreatedAt int64
	Kind      uint32
	Tags      []Tag
	Content   string
}

func (id EventID) String() string    { return hex.EncodeToString(id[:]) }
func (key PublicKey) String() string { return hex.EncodeToString(key[:]) }
func (sig Signature) String() string { return hex.EncodeToString(sig[:]) }

// ParseEventID parses a lower- or upper-case hexadecimal event ID.
func ParseEventID(value string) (EventID, error) {
	decoded, err := parseFixedHex(value, eventIDSize, "event id")
	if err != nil {
		return EventID{}, err
	}
	var result EventID
	copy(result[:], decoded)
	return result, nil
}

// ParsePublicKey parses a lower- or upper-case hexadecimal x-only public key.
func ParsePublicKey(value string) (PublicKey, error) {
	decoded, err := parseFixedHex(value, publicKeySize, "public key")
	if err != nil {
		return PublicKey{}, err
	}
	var result PublicKey
	copy(result[:], decoded)
	return result, nil
}

// ParseSignature parses a lower- or upper-case hexadecimal Schnorr signature.
func ParseSignature(value string) (Signature, error) {
	decoded, err := parseFixedHex(value, signatureSize, "signature")
	if err != nil {
		return Signature{}, err
	}
	var result Signature
	copy(result[:], decoded)
	return result, nil
}

func parseFixedHex(value string, want int, field string) ([]byte, error) {
	if len(value) != want*2 {
		return nil, fmt.Errorf("%w: %s must be %d hexadecimal characters", ErrInvalidIdentifier, field, want*2)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidIdentifier, field, err)
	}
	return decoded, nil
}

// CanonicalBytes returns the NIP-01 canonical serialization used to calculate
// an event ID. Protobuf bytes never participate in this calculation.
func (e Event) CanonicalBytes(limits Limits) ([]byte, error) {
	if err := e.ValidateShape(limits); err != nil {
		return nil, err
	}
	buf := make([]byte, 0, 96+len(e.Content))
	buf = append(buf, "[0,"...)
	buf = appendJSONString(buf, e.PublicKey.String())
	buf = append(buf, ',')
	buf = strconv.AppendInt(buf, e.CreatedAt, 10)
	buf = append(buf, ',')
	buf = strconv.AppendUint(buf, uint64(e.Kind), 10)
	buf = append(buf, ',')
	buf = append(buf, '[')
	for i, tag := range e.Tags {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, '[')
		for j, value := range tag {
			if j > 0 {
				buf = append(buf, ',')
			}
			buf = appendJSONString(buf, value)
		}
		buf = append(buf, ']')
	}
	buf = append(buf, ']', ',')
	buf = appendJSONString(buf, e.Content)
	buf = append(buf, ']')
	return buf, nil
}

// appendJSONString follows the byte-oriented escaping used by NIP-01's
// JSON.stringify serialization. In particular, printable UTF-8 bytes remain
// unchanged and control characters use JSON short escapes or four hex digits.
func appendJSONString(dst []byte, value string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c == '"':
			dst = append(dst, '\\', '"')
		case c == '\\':
			dst = append(dst, '\\', '\\')
		case c >= 0x20:
			dst = append(dst, c)
		case c == '\b':
			dst = append(dst, '\\', 'b')
		case c == '\t':
			dst = append(dst, '\\', 't')
		case c == '\n':
			dst = append(dst, '\\', 'n')
		case c == '\f':
			dst = append(dst, '\\', 'f')
		case c == '\r':
			dst = append(dst, '\\', 'r')
		default:
			dst = append(dst, '\\', 'u', '0', '0', hexDigit(c>>4), hexDigit(c&0x0f))
		}
	}
	return append(dst, '"')
}

func hexDigit(value byte) byte {
	if value < 10 {
		return '0' + value
	}
	return 'a' + value - 10
}

// ComputeID calculates the SHA-256 NIP-01 identifier.
func (e Event) ComputeID(limits Limits) (EventID, error) {
	canonical, err := e.CanonicalBytes(limits)
	if err != nil {
		return EventID{}, err
	}
	return sha256.Sum256(canonical), nil
}

// ValidateShape verifies bounded event structure, its canonical ID and the
// BIP-340 signature. It does not apply host policy or authorization.
func (e Event) Validate(limits Limits) error {
	computed, err := e.ComputeID(limits)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(computed[:], e.ID[:]) != 1 {
		return invalidEvent("id", "does not match canonical NIP-01 serialization")
	}
	publicKey, err := schnorr.ParsePubKey(e.PublicKey[:])
	if err != nil {
		return invalidEvent("public_key", "is not a valid BIP-340 key")
	}
	signature, err := schnorr.ParseSignature(e.Signature[:])
	if err != nil {
		return invalidEvent("signature", "is not a valid BIP-340 signature")
	}
	if !signature.Verify(e.ID[:], publicKey) {
		return invalidEvent("signature", "does not verify event id")
	}
	return nil
}

// ValidateShape validates bounded event fields that do not require crypto.
func (e Event) ValidateShape(limits Limits) error {
	limits = limits.WithDefaults()
	if err := limits.Validate(); err != nil {
		return err
	}
	if len(e.Tags) > limits.MaxTagCount {
		return invalidEvent("tags", "exceeds tag count limit")
	}
	if len(e.Content) > limits.MaxEventBytes {
		return invalidEvent("content", "exceeds event size limit")
	}
	for _, tag := range e.Tags {
		if len(tag) > limits.MaxTagValues {
			return invalidEvent("tag", "exceeds values limit")
		}
		for _, value := range tag {
			if len(value) > limits.MaxTagValueBytes {
				return invalidEvent("tag value", "exceeds value size limit")
			}
		}
	}
	canonical, err := e.canonicalBytesUnchecked()
	if err != nil {
		return err
	}
	if len(canonical) > limits.MaxEventBytes {
		return invalidEvent("event", "exceeds event size limit")
	}
	return nil
}

func (e Event) canonicalBytesUnchecked() ([]byte, error) {
	buf := make([]byte, 0, 96+len(e.Content))
	buf = append(buf, "[0,"...)
	buf = strconv.AppendQuote(buf, e.PublicKey.String())
	buf = append(buf, ',')
	buf = strconv.AppendInt(buf, e.CreatedAt, 10)
	buf = append(buf, ',')
	buf = strconv.AppendUint(buf, uint64(e.Kind), 10)
	buf = append(buf, ',', '[')
	for i, tag := range e.Tags {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, '[')
		for j, value := range tag {
			if j > 0 {
				buf = append(buf, ',')
			}
			buf = strconv.AppendQuote(buf, value)
		}
		buf = append(buf, ']')
	}
	buf = append(buf, ']', ',')
	buf = strconv.AppendQuote(buf, e.Content)
	buf = append(buf, ']')
	return buf, nil
}
