package migratestrfry

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

type batchPersister interface {
	Persist(context.Context, []*nostr.Event) (persistResult, error)
}

type persistResult struct {
	Persisted int
	Rejected  int
}

type Result struct {
	Read      int
	Valid     int
	Persisted int
	Invalid   int
	Rejected  int
}

func (r Result) Errors() int {
	return r.Invalid + r.Rejected
}

func processExport(
	ctx context.Context,
	reader io.Reader,
	options CLIOptions,
	persister batchPersister,
) (Result, error) {
	processor, err := newProcessor(options, persister)
	if err != nil {
		return Result{}, err
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxExportLineBytes)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if err := processor.AddLine(ctx, lineNumber, scanner.Bytes()); err != nil {
			return processor.result, err
		}
	}
	if err := scanner.Err(); err != nil {
		return processor.result, fmt.Errorf("read exported JSONL: %w", err)
	}

	return processor.Finish(ctx)
}

type processor struct {
	options   CLIOptions
	persister batchPersister
	batch     []*nostr.Event
	result    Result
}

func newProcessor(options CLIOptions, persister batchPersister) (*processor, error) {
	if !options.DryRun && persister == nil {
		return nil, errors.New("event persister cannot be nil outside dry-run mode")
	}

	return &processor{
		options:   options,
		persister: persister,
		batch:     make([]*nostr.Event, 0, options.BatchSize),
	}, nil
}

func (p *processor) AddLine(ctx context.Context, _ int, line []byte) error {
	if len(line) == 0 {
		return nil
	}
	p.result.Read++

	event, err := parseEvent(line)
	if err != nil {
		p.result.Invalid++
		return nil
	}
	p.result.Valid++
	p.batch = append(p.batch, event)
	if len(p.batch) < p.options.BatchSize {
		return nil
	}
	if err := persistBatch(ctx, p.batch, p.options.DryRun, p.persister, &p.result); err != nil {
		return err
	}
	p.batch = make([]*nostr.Event, 0, p.options.BatchSize)
	return nil
}

func (p *processor) Finish(ctx context.Context) (Result, error) {
	if len(p.batch) > 0 {
		if err := persistBatch(ctx, p.batch, p.options.DryRun, p.persister, &p.result); err != nil {
			return p.result, err
		}
	}
	if p.result.Errors() > 0 && p.options.FailOnError {
		return p.result, fmt.Errorf("strfry migration completed with %d row errors", p.result.Errors())
	}

	return p.result, nil
}

func parseEvent(line []byte) (*nostr.Event, error) {
	var event nostr.Event
	if err := json.Unmarshal(line, &event); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}
	if !event.CheckID() {
		return nil, errors.New("event ID does not match its contents")
	}
	ok, err := event.CheckSignature()
	if err != nil {
		return nil, fmt.Errorf("check event signature: %w", err)
	}
	if !ok {
		return nil, errors.New("invalid event signature")
	}

	return &event, nil
}

func persistBatch(
	ctx context.Context,
	batch []*nostr.Event,
	dryRun bool,
	persister batchPersister,
	result *Result,
) error {
	if dryRun {
		return nil
	}
	persisted, err := persister.Persist(ctx, batch)
	if err != nil {
		return fmt.Errorf("persist export batch: %w", err)
	}
	result.Persisted += persisted.Persisted
	result.Rejected += persisted.Rejected
	return nil
}
