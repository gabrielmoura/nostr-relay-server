package jsonx

import (
	"bytes"
	stdjson "encoding/json"

	"github.com/bytedance/sonic"
)

type RawMessage = stdjson.RawMessage
type NoCopyRawMessage = stdjson.RawMessage

var sonicStd = sonic.ConfigStd

func Marshal(v any) ([]byte, error) {
	return sonicStd.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	switch v.(type) {
	case *any, *map[string]any, *[]any:
		return unmarshalUseNumber(data, v)
	}

	if err := sonicStd.Unmarshal(data, v); err == nil {
		return nil
	}

	return unmarshalUseNumber(data, v)
}

func unmarshalUseNumber(data []byte, v any) error {
	decoder := stdjson.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(v)
}

func NewEncoder(w interface{ Write([]byte) (int, error) }) *stdjson.Encoder {
	return stdjson.NewEncoder(w)
}

func NewDecoder(r interface{ Read([]byte) (int, error) }) *stdjson.Decoder {
	decoder := stdjson.NewDecoder(r)
	decoder.UseNumber()
	return decoder
}
