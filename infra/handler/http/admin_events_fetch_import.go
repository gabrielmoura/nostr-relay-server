package http

import (
	"bufio"
	"context"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func FetchEventFromRelays() fiber.Handler {
	return func(c *fiber.Ctx) error {
		eventID := strings.ToLower(strings.TrimSpace(c.Params("id")))
		if !eventIDPattern.MatchString(eventID) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
		}

		var req adminFetchEventRequest
		if len(c.Body()) > 0 {
			if err := parseAdminJSONBody(c, &req); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
			}
		}

		relays := mergeFetchRelayList(req.Relays, config.Cfg.Stream.Relays, defaultAdminFetchRelays)
		event, sourceRelay, tried, relayResults, err := fetchEventFromRelays(c.UserContext(), eventID, relays)
		if err != nil {
			if errors.Is(err, errAdminEventNotFoundOnRelays) {
				return c.JSON(adminFetchEventResponse{EventID: eventID, Found: false, Persisted: false, RelaysTried: tried, RelayResults: relayResults, Message: "event not found on provided relays"})
			}
			return internalServerError(c, err)
		}

		persisted := true
		if err := db.DbQueries.InsertEvent(c.UserContext(), event); err != nil {
			if errors.Is(err, dbmodel.ErrDupEvent) {
				persisted = false
			} else {
				return internalServerError(c, err)
			}
		}

		return c.JSON(adminFetchEventResponse{EventID: event.ID, SourceRelay: sourceRelay, Found: true, Persisted: persisted, RelaysTried: tried, RelayResults: relayResults, Message: "event found on relay"})
	}
}

func ImportEventsJSONL() fiber.Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid multipart form"})
		}

		files := form.File["files"]
		if len(files) == 0 {
			files = form.File["file"]
		}
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no files provided"})
		}

		results := make([]adminImportFileResult, 0, len(files))
		for _, fileHeader := range files {
			results = append(results, processImportFile(c, fileHeader))
		}

		return c.JSON(adminImportEventsResponse{Files: results})
	}
}

func processImportFile(c *fiber.Ctx, fileHeader *multipart.FileHeader) adminImportFileResult {
	tmpFile, err := os.CreateTemp("", "nostr-admin-import-*.jsonl")
	if err != nil {
		return adminImportFileResult{Filename: fileHeader.Filename, Error: err.Error()}
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := c.SaveFile(fileHeader, tmpPath); err != nil {
		return adminImportFileResult{Filename: fileHeader.Filename, Error: err.Error()}
	}

	file, err := os.Open(filepath.Clean(tmpPath))
	if err != nil {
		return adminImportFileResult{Filename: fileHeader.Filename, Error: err.Error()}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	result := adminImportFileResult{Filename: fileHeader.Filename}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		result.Total++

		var event nostr.Event
		if err := event.UnmarshalJSON([]byte(line)); err != nil || !event.CheckID() {
			result.Invalid++
			continue
		}
		ok, sigErr := event.CheckSignature()
		if sigErr != nil || !ok {
			result.Invalid++
			continue
		}

		err := db.DbQueries.InsertEvent(c.UserContext(), &event)
		switch {
		case err == nil:
			result.Inserted++
		case errors.Is(err, dbmodel.ErrDupEvent):
			result.Duplicates++
		default:
			result.Error = err.Error()
			return result
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && result.Error == "" {
		result.Error = scanErr.Error()
	}

	return result
}

func fetchEventFromRelays(ctx context.Context, eventID string, relays []string) (*nostr.Event, string, int, []adminFetchRelayResult, error) {
	if len(relays) == 0 {
		return nil, "", 0, []adminFetchRelayResult{}, errAdminEventNotFoundOnRelays
	}

	results := make([]adminFetchRelayResult, 0, len(relays))
	for _, relayURL := range relays {
		relayCtx, cancelRelay := context.WithTimeout(ctx, 5*time.Second)
		relay, err := nostr.RelayConnect(relayCtx, relayURL)
		if err != nil {
			cancelRelay()
			results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "connect_error", Error: err.Error()})
			continue
		}

		events, err := relay.QuerySync(relayCtx, nostr.Filter{IDs: []string{eventID}, Limit: 1})
		_ = relay.Close()
		cancelRelay()
		if err != nil {
			results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "query_error", Error: err.Error()})
			continue
		}
		if len(events) == 0 {
			results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "not_found"})
			continue
		}

		results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "found"})
		return events[0], relayURL, len(results), results, nil
	}

	return nil, "", len(results), results, errAdminEventNotFoundOnRelays
}
