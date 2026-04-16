package protocol

import (
	"fmt"
)

type byteBuffer struct {
	data []byte
	pos  int
}

func newBuffer(data []byte) *byteBuffer {
	return &byteBuffer{data: data}
}

func (b *byteBuffer) Empty() bool {
	return b.pos >= len(b.data)
}

func (b *byteBuffer) ReadByte() (byte, error) {
	if b.Empty() {
		return 0, ErrUnexpectedEOF
	}

	value := b.data[b.pos]
	b.pos++

	return value, nil
}

func (b *byteBuffer) ReadN(n int) ([]byte, error) {
	if n < 0 || b.pos+n > len(b.data) {
		return nil, ErrUnexpectedEOF
	}

	v := b.data[b.pos : b.pos+n]
	b.pos += n

	return v, nil
}

func decodeVarInt(b *byteBuffer) (uint64, error) {
	var out uint64

	for {
		c, err := b.ReadByte()
		if err != nil {
			return 0, err
		}

		out = (out << 7) | uint64(c&0x7F)
		if c&0x80 == 0 {
			break
		}
	}

	return out, nil
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

func decodeBound(b *byteBuffer, lastTimestamp uint64) (Bound, uint64, error) {
	encodedTimestamp, err := decodeVarInt(b)
	if err != nil {
		return Bound{}, 0, err
	}

	timestamp := decodeTimestamp(encodedTimestamp, lastTimestamp)

	prefixLen, err := decodeVarInt(b)
	if err != nil {
		return Bound{}, 0, err
	}

	if prefixLen > 32 {
		return Bound{}, 0, fmt.Errorf("invalid prefix size: %d", prefixLen)
	}

	prefix, err := b.ReadN(int(prefixLen))
	if err != nil {
		return Bound{}, 0, err
	}

	bnd := Bound{Timestamp: timestamp, Prefix: make([]byte, len(prefix))}
	copy(bnd.Prefix, prefix)

	return bnd, timestamp, nil
}

func encodeBound(bound Bound, lastTimestamp uint64) ([]byte, uint64) {
	out := make([]byte, 0, 16+len(bound.Prefix))
	timestampPart, nextTimestamp := encodeTimestamp(bound.Timestamp, lastTimestamp)
	out = append(out, timestampPart...)
	out = append(out, encodeVarInt(uint64(len(bound.Prefix)))...)
	out = append(out, bound.Prefix...)

	return out, nextTimestamp
}

func decodeTimestamp(encoded uint64, lastTimestamp uint64) uint64 {
	if encoded == 0 {
		return Infinity
	}

	offset := encoded - 1
	if lastTimestamp == Infinity {
		return Infinity
	}

	if Infinity-lastTimestamp < offset {
		return Infinity
	}

	return lastTimestamp + offset
}

func encodeTimestamp(timestamp uint64, lastTimestamp uint64) ([]byte, uint64) {
	if timestamp == Infinity {
		return encodeVarInt(0), Infinity
	}

	if lastTimestamp == Infinity {
		lastTimestamp = 0
	}

	delta := timestamp - lastTimestamp

	return encodeVarInt(delta + 1), timestamp
}
