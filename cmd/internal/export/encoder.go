package export

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

type fileEncoder interface {
	Write(*nostr.Event) error
	Flush() error
}

type jsonlFileEncoder struct {
	w Writer
}

func (e *jsonlFileEncoder) Write(event *nostr.Event) error {
	return e.w.Write(event)
}

func (e *jsonlFileEncoder) Flush() error {
	return nil
}

type tsvFileEncoder struct {
	cw *csv.Writer
}

func (e *tsvFileEncoder) Write(event *nostr.Event) error {
	tags, err := json.Marshal(event.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags for event %s: %w", event.ID, err)
	}

	record := []string{
		event.ID,
		event.PubKey,
		strconv.FormatInt(event.CreatedAt.Time().Unix(), 10),
		strconv.Itoa(event.Kind),
		string(tags),
		event.Content,
		event.Sig,
	}

	if err := e.cw.Write(record); err != nil {
		return fmt.Errorf("write tsv event %s: %w", event.ID, err)
	}

	return nil
}

func (e *tsvFileEncoder) Flush() error {
	e.cw.Flush()
	if err := e.cw.Error(); err != nil {
		return fmt.Errorf("flush tsv writer: %w", err)
	}
	return nil
}

func newFileEncoder(format OutputFormat, w io.Writer) (fileEncoder, error) {
	switch format {
	case FormatJSONL:
		return &jsonlFileEncoder{w: NewWriter(w)}, nil
	case FormatTSV:
		cw := csv.NewWriter(w)
		cw.Comma = '\t'
		return &tsvFileEncoder{cw: cw}, nil
	default:
		return nil, fmt.Errorf("unsupported export format %q", format)
	}
}

func writeTSVHeader(buf *bufio.Writer) error {
	_, err := buf.WriteString("id\tpubkey\tcreated_at\tkind\ttags\tcontent\tsig\n")
	if err != nil {
		return fmt.Errorf("write tsv header: %w", err)
	}
	return nil
}
