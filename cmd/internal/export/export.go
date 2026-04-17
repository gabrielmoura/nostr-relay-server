package export

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func Export(opt *Options) error {
	if opt == nil {
		return fmt.Errorf("export options cannot be nil")
	}

	if err := setupEnvironment(); err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	filterPayload, err := resolveFilterPayload(opt.Filter, opt.FilterFile)
	if err != nil {
		return err
	}

	filter, err := parseFilter(filterPayload)
	if err != nil {
		return err
	}

	writer, err := newSegmentWriter(opt)
	if err != nil {
		return err
	}
	defer writer.Close()

	total, err := exportEvents(ctx, opt, filter, writer)
	if err != nil {
		return err
	}

	log.Logger.Info(
		"export completed",
		zap.Int64("total_exported", total),
		zap.String("base_file", opt.Filename),
		zap.String("format", string(opt.Format)),
		zap.Int("segments", writer.filesCreated),
	)
	return nil
}

func setupEnvironment() error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Init()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Init(ctx); err != nil {
		return fmt.Errorf("init database: %w", err)
	}

	return nil
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Logger.Info("received stop signal, shutting down export")
		cancel()
	}()

	return ctx, cancel
}

func parseFilter(raw string) (nostr.Filter, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nostr.Filter{}, nil
	}

	var probe any
	if err := json.Unmarshal([]byte(value), &probe); err != nil {
		return nostr.Filter{}, fmt.Errorf("invalid --filter JSON: %w", err)
	}

	if _, ok := probe.(map[string]any); !ok {
		return nostr.Filter{}, fmt.Errorf("invalid --filter JSON: expected object")
	}

	var filter nostr.Filter
	if err := json.Unmarshal([]byte(value), &filter); err != nil {
		return nostr.Filter{}, fmt.Errorf("invalid --filter payload: %w", err)
	}

	return filter, nil
}

func resolveFilterPayload(filter string, filterFile string) (string, error) {
	inline := strings.TrimSpace(filter)
	path := strings.TrimSpace(filterFile)

	if inline != "" && path != "" {
		return "", fmt.Errorf("invalid filter source: use only one of --filter or --filter-file")
	}

	if path == "" {
		return inline, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read --filter-file %q: %w", path, err)
	}

	return strings.TrimSpace(string(content)), nil
}

func exportEvents(ctx context.Context, opt *Options, filter nostr.Filter, writer *segmentWriter) (int64, error) {
	if opt.Filter == "" {
		return exportAllEvents(ctx, opt, writer)
	}

	return exportFilteredEvents(ctx, opt, filter, writer)
}

func exportAllEvents(ctx context.Context, opt *Options, writer *segmentWriter) (int64, error) {
	total := int64(0)

	for batch := range db.DbQueries.StreamAllEvents(ctx, opt.BatchSize) {
		for i := range *batch {
			if opt.Limit > 0 && int(total) >= opt.Limit {
				return total, nil
			}

			if err := writer.WriteEvent(&(*batch)[i]); err != nil {
				return total, err
			}
			total++
		}
	}

	return total, nil
}

func exportFilteredEvents(ctx context.Context, opt *Options, filter nostr.Filter, writer *segmentWriter) (int64, error) {
	total := int64(0)
	offset := 0

	for {
		pageLimit := opt.BatchSize
		if opt.Limit > 0 {
			remaining := opt.Limit - int(total)
			if remaining <= 0 {
				return total, nil
			}
			if remaining < pageLimit {
				pageLimit = remaining
			}
		}

		pageFilter := filter.Clone()
		pageFilter.Limit = pageLimit
		pageFilter.LimitZero = false

		events, err := db.DbQueries.QueryEventsPage(ctx, pageFilter, offset)
		if err != nil {
			return total, fmt.Errorf("query filtered events: %w", err)
		}
		if len(events) == 0 {
			break
		}

		for _, event := range events {
			if err := writer.WriteEvent(event); err != nil {
				return total, err
			}
			total++
		}

		offset += len(events)
		if len(events) < pageLimit {
			break
		}
	}

	return total, nil
}

type segmentWriter struct {
	options      *Options
	basePath     string
	baseNoExt    string
	ext          string
	currentIndex int
	currentCount int
	filesCreated int

	file    *os.File
	buffer  *bufio.Writer
	encoder fileEncoder
}

func newSegmentWriter(options *Options) (*segmentWriter, error) {
	basePath := options.Filename
	ext := filepath.Ext(basePath)
	if ext == "" {
		ext = "." + string(options.Format)
		basePath += ext
	}

	return &segmentWriter{
		options:   options,
		basePath:  basePath,
		baseNoExt: strings.TrimSuffix(basePath, ext),
		ext:       ext,
	}, nil
}

func (s *segmentWriter) WriteEvent(event *nostr.Event) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}

	if s.options.SegmentSize > 0 && s.currentCount >= s.options.SegmentSize {
		if err := s.rotate(); err != nil {
			return err
		}
	}

	if err := s.encoder.Write(event); err != nil {
		return fmt.Errorf("encode event %s: %w", event.ID, err)
	}

	s.currentCount++
	return nil
}

func (s *segmentWriter) Close() error {
	if s.encoder != nil {
		if err := s.encoder.Flush(); err != nil {
			return err
		}
	}
	if s.buffer != nil {
		if err := s.buffer.Flush(); err != nil {
			return err
		}
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *segmentWriter) ensureOpen() error {
	if s.file != nil {
		return nil
	}
	return s.openNextFile()
}

func (s *segmentWriter) rotate() error {
	if err := s.Close(); err != nil {
		return err
	}
	s.file = nil
	s.buffer = nil
	s.encoder = nil
	s.currentCount = 0
	return s.openNextFile()
}

func (s *segmentWriter) openNextFile() error {
	s.currentIndex++
	path := s.currentPath()

	if !s.options.Overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("output file %q already exists (use --overwrite)", path)
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open export output %q: %w", path, err)
	}

	buffer := bufio.NewWriter(file)
	encoder, err := newFileEncoder(s.options.Format, buffer)
	if err != nil {
		_ = file.Close()
		return err
	}

	if s.options.Format == FormatTSV && !s.options.NoHeader {
		if err := writeTSVHeader(buffer); err != nil {
			_ = file.Close()
			return err
		}
	}

	s.file = file
	s.buffer = buffer
	s.encoder = encoder
	s.currentCount = 0
	s.filesCreated++

	log.Logger.Info("opened export file", zap.String("path", path))
	return nil
}

func (s *segmentWriter) currentPath() string {
	if s.options.SegmentSize <= 0 {
		return s.basePath
	}

	return fmt.Sprintf("%s-%03d%s", s.baseNoExt, s.currentIndex, s.ext)
}
