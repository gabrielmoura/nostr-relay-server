package relaysync

import (
	"errors"
	"strings"

	"github.com/klauspost/compress/zstd"
)

var ErrDecompressedLimit = errors.New("zstd output exceeds configured limit")

type ZstdOptions struct {
	Level     int
	Threshold int
	MaxOutput int
}

type ZstdCodec struct {
	options ZstdOptions
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func NewZstdCodec(options ZstdOptions) (*ZstdCodec, error) {
	if options.Level == 0 {
		options.Level = 1
	}
	if options.Threshold < 0 || options.MaxOutput <= 0 {
		return nil, ErrInvalidLimits
	}
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(options.Level)))
	if err != nil {
		return nil, err
	}
	decoder, err := zstd.NewReader(nil, zstd.WithDecodeAllCapLimit(true))
	if err != nil {
		encoder.Close()
		return nil, err
	}
	return &ZstdCodec{options: options, encoder: encoder, decoder: decoder}, nil
}

func (c *ZstdCodec) Close() error {
	if c == nil {
		return nil
	}
	if c.encoder != nil {
		c.encoder.Close()
	}
	if c.decoder != nil {
		c.decoder.Close()
	}
	return nil
}

func (c *ZstdCodec) Encode(payload []byte) ([]byte, bool, error) {
	if c == nil || len(payload) < c.options.Threshold {
		return append([]byte(nil), payload...), false, nil
	}
	encoded := c.encoder.EncodeAll(payload, nil)
	if len(encoded) >= len(payload) {
		return append([]byte(nil), payload...), false, nil
	}
	return encoded, true, nil
}

func (c *ZstdCodec) Decode(payload []byte) ([]byte, error) {
	if c == nil {
		return append([]byte(nil), payload...), nil
	}
	decoded, err := c.decoder.DecodeAll(payload, make([]byte, 0, c.options.MaxOutput+1))
	if err != nil {
		if strings.Contains(err.Error(), "decompressed size exceeds configured limit") {
			return nil, ErrDecompressedLimit
		}
		return nil, err
	}
	if len(decoded) > c.options.MaxOutput {
		return nil, ErrDecompressedLimit
	}
	return decoded, nil
}
