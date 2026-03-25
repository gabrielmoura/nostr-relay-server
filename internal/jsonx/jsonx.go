package jsonx

import (
	"bytes"
	stdjson "encoding/json"
)

type RawMessage = stdjson.RawMessage
type NoCopyRawMessage = stdjson.RawMessage

func Marshal(v any) ([]byte, error) {
	return stdjson.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
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
