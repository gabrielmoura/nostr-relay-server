package graph

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gofiber/fiber/v2"
)

type adminRequest struct {
	method      string
	route       string
	path        string
	query       url.Values
	body        any
	uploads     []*graphql.Upload
	handlerFunc fiber.Handler
}

func executeAdminRequest(ctx context.Context, req adminRequest) ([]byte, error) {
	app := fiber.New(fiber.Config{JSONEncoder: jsonx.Marshal, JSONDecoder: jsonx.Unmarshal})
	app.Add(req.method, req.route, req.handlerFunc)

	body, contentType, err := adminRequestBody(req)
	if err != nil {
		return nil, err
	}

	target := req.path
	if len(req.query) > 0 {
		target += "?" + req.query.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, target, body)
	if err != nil {
		return nil, err
	}
	httpReq.Host = "admin.local"
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, err := app.Test(httpReq, int((30 * time.Second).Milliseconds()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, adminErrorFromPayload(resp.StatusCode, payload)
	}

	return payload, nil
}

func adminRequestBody(req adminRequest) (io.Reader, string, error) {
	if len(req.uploads) > 0 {
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)
		for _, upload := range req.uploads {
			if upload == nil || upload.File == nil {
				continue
			}
			if _, err := upload.File.Seek(0, io.SeekStart); err != nil {
				return nil, "", err
			}
			part, err := writer.CreateFormFile("files", upload.Filename)
			if err != nil {
				return nil, "", err
			}
			if _, err := io.Copy(part, upload.File); err != nil {
				return nil, "", err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buffer, writer.FormDataContentType(), nil
	}

	if req.body == nil {
		return nil, "", nil
	}

	payload, err := jsonx.Marshal(req.body)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(payload), fiber.MIMEApplicationJSON, nil
}

func adminErrorFromPayload(statusCode int, payload []byte) error {
	var body map[string]any
	if err := jsonx.Unmarshal(payload, &body); err == nil {
		if message, ok := body["error"].(string); ok && strings.TrimSpace(message) != "" {
			return fmt.Errorf("admin request failed (%d): %s", statusCode, message)
		}
	}
	return fmt.Errorf("admin request failed (%d)", statusCode)
}

func decodeRESTModel[T any](payload []byte) (*T, error) {
	normalized, err := normalizeRESTPayload(payload)
	if err != nil {
		return nil, err
	}
	var out T
	if err := jsonx.Unmarshal(normalized, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func decodeRESTAny(payload []byte) (any, error) {
	normalized, err := normalizeRESTPayload(payload)
	if err != nil {
		return nil, err
	}
	var out any
	if err := jsonx.Unmarshal(normalized, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeRESTList[T any](payload []byte) ([]*T, error) {
	normalized, err := normalizeRESTPayload(payload)
	if err != nil {
		return nil, err
	}
	var out []*T
	if err := jsonx.Unmarshal(normalized, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type restPageEnvelope struct {
	Items   stdjson.RawMessage `json:"items"`
	Total   int32              `json:"total"`
	Limit   int32              `json:"limit"`
	Offset  int32              `json:"offset"`
	HasMore bool               `json:"has_more"`
}

type restItemsEnvelope struct {
	Items stdjson.RawMessage `json:"items"`
}

func decodeRESTPage[T any](payload []byte) ([]*T, *model.PageInfo, error) {
	var envelope restPageEnvelope
	if err := jsonx.Unmarshal(payload, &envelope); err != nil {
		return nil, nil, err
	}
	items, err := decodeRESTList[T](envelope.Items)
	if err != nil {
		return nil, nil, err
	}
	return items, &model.PageInfo{Total: envelope.Total, Limit: envelope.Limit, Offset: envelope.Offset, HasMore: envelope.HasMore}, nil
}

func decodeRESTMapPage(payload []byte) ([]map[string]any, *model.PageInfo, error) {
	var envelope restPageEnvelope
	if err := jsonx.Unmarshal(payload, &envelope); err != nil {
		return nil, nil, err
	}
	normalized, err := normalizeRESTPayload(envelope.Items)
	if err != nil {
		return nil, nil, err
	}
	items := make([]map[string]any, 0)
	if err := jsonx.Unmarshal(normalized, &items); err != nil {
		return nil, nil, err
	}
	return items, &model.PageInfo{Total: envelope.Total, Limit: envelope.Limit, Offset: envelope.Offset, HasMore: envelope.HasMore}, nil
}

func decodeRESTItems[T any](payload []byte) ([]*T, error) {
	var envelope restItemsEnvelope
	if err := jsonx.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	return decodeRESTList[T](envelope.Items)
}

func paginatePointers[T any](items []*T, page *model.OffsetPageInput) ([]*T, *model.PageInfo) {
	limit := int32PtrValue(nil, 100)
	offset := int32(0)
	if page != nil {
		limit = int32PtrValue(page.Limit, 100)
		offset = int32PtrValue(page.Offset, 0)
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	total := int32(len(items))
	if offset >= total {
		return []*T{}, &model.PageInfo{Total: total, Limit: limit, Offset: offset, HasMore: false}
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], &model.PageInfo{Total: total, Limit: limit, Offset: offset, HasMore: end < total}
}

func paginateMaps(items []map[string]any, page *model.OffsetPageInput) ([]map[string]any, *model.PageInfo) {
	limit := int32PtrValue(nil, 100)
	offset := int32(0)
	if page != nil {
		limit = int32PtrValue(page.Limit, 100)
		offset = int32PtrValue(page.Offset, 0)
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	total := int32(len(items))
	if offset >= total {
		return []map[string]any{}, &model.PageInfo{Total: total, Limit: limit, Offset: offset, HasMore: false}
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], &model.PageInfo{Total: total, Limit: limit, Offset: offset, HasMore: end < total}
}

func int32PtrValue(value *int32, fallback int32) int32 {
	if value == nil {
		return fallback
	}
	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func adminPageQuery(page *model.OffsetPageInput) url.Values {
	values := url.Values{}
	if page == nil {
		return values
	}
	if page.Limit != nil {
		values.Set("limit", fmt.Sprintf("%d", *page.Limit))
	}
	if page.Offset != nil {
		values.Set("offset", fmt.Sprintf("%d", *page.Offset))
	}
	return values
}

func adminMutationAck(ok bool, entityID *string, message string) *model.MutationAck {
	ack := &model.MutationAck{Ok: ok, EntityID: entityID}
	if strings.TrimSpace(message) != "" {
		ack.Message = &message
	}
	return ack
}
