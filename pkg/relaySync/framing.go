package relaysync

import (
	"encoding/binary"
	"errors"
)

const relaySyncFrameHeaderSize = 8

var (
	ErrInvalidFrame  = errors.New("invalid RelaySync frame")
	ErrFrameTooLarge = errors.New("RelaySync frame exceeds limit")
)

type FrameOptions struct{ MaxPayload int }

func EncodeFrame(payload []byte, originalSize int, compressed bool, options FrameOptions) ([]byte, error) {
	if options.MaxPayload <= 0 || originalSize < 0 || originalSize > options.MaxPayload || len(payload) > options.MaxPayload {
		return nil, ErrFrameTooLarge
	}
	frame := make([]byte, relaySyncFrameHeaderSize+len(payload))
	frame[0], frame[1], frame[2] = 'R', 'S', 2
	if compressed {
		frame[3] = 1
	}
	binary.BigEndian.PutUint32(frame[4:8], uint32(originalSize))
	copy(frame[8:], payload)
	return frame, nil
}

func DecodeFrame(frame []byte, options FrameOptions) (payload []byte, compressed bool, err error) {
	if len(frame) < relaySyncFrameHeaderSize || frame[0] != 'R' || frame[1] != 'S' || frame[2] != 2 || frame[3]&0xfe != 0 {
		return nil, false, ErrInvalidFrame
	}
	size := int(binary.BigEndian.Uint32(frame[4:8]))
	if options.MaxPayload <= 0 || size > options.MaxPayload {
		return nil, false, ErrFrameTooLarge
	}
	if frame[3]&1 == 0 && size != len(frame)-relaySyncFrameHeaderSize {
		return nil, false, ErrInvalidFrame
	}
	return append([]byte(nil), frame[8:]...), frame[3]&1 != 0, nil
}
