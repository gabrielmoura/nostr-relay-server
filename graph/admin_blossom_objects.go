package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
)

type blossomObjectListSource interface {
	ListBlossomObjects(context.Context, dbstore.BlossomObjectFilters, int, int) ([]dbstore.BlossomObjectRow, int64, error)
}

type blossomObjectSummarySource interface {
	ListBlossomObjectSummaries(context.Context, dbstore.BlossomObjectFilters, int, int) ([]dbstore.BlossomObjectRow, int64, error)
}

func (r *Resolver) queryBlossomObjects(ctx context.Context, filter *model.AdminBlossomObjectFilterInput, page *model.OffsetPageInput) (*model.AdminBlossomObjectPage, error) {
	source := r.blossomObjectSource
	if source == nil {
		if internaldb.DbQueries == nil {
			return nil, errors.New("blossom query service is not initialized")
		}
		source = internaldb.DbQueries
	}
	filters := dbstore.BlossomObjectFilters{}
	if filter != nil {
		filters = dbstore.BlossomObjectFilters{
			Query: strings.TrimSpace(stringValue(filter.Q)), SHA256: strings.TrimSpace(stringValue(filter.Sha256)),
			MIMEType: strings.TrimSpace(stringValue(filter.MimeType)), Extension: strings.TrimSpace(stringValue(filter.Extension)),
			ReviewState: strings.TrimSpace(stringValue(filter.ReviewState)), Pubkey: strings.TrimSpace(stringValue(filter.Pubkey)),
			UploaderQuery: strings.TrimSpace(stringValue(filter.UploaderQuery)),
		}
	}
	limit, offset := loggedUsersPage(page)
	var rows []dbstore.BlossomObjectRow
	var total int64
	var err error
	if summary, ok := source.(blossomObjectSummarySource); ok && blossomObjectSummarySelected(ctx) {
		rows, total, err = summary.ListBlossomObjectSummaries(ctx, filters, limit, offset)
	} else {
		rows, total, err = source.ListBlossomObjects(ctx, filters, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	items := make([]*model.AdminBlossomObject, 0, len(rows))
	for _, row := range rows {
		items = append(items, blossomObjectModel(row))
	}
	return &model.AdminBlossomObjectPage{Items: items, PageInfo: &model.PageInfo{
		Total: int32(min(total, math.MaxInt32)), Limit: int32(limit), Offset: int32(offset), HasMore: int64(offset)+int64(len(rows)) < total,
	}}, nil
}

// Only select the reduced database projection when every requested object
// field is available from it; arbitrary GraphQL callers can request details.
func blossomObjectSummarySelected(ctx context.Context) bool {
	field, operation := graphql.GetFieldContext(ctx), graphql.GetOperationContext(ctx)
	if field == nil || operation == nil {
		return false
	}
	for _, child := range graphql.CollectFields(operation, field.Field.Selections, nil) {
		if child.Name != "items" {
			continue
		}
		for _, item := range graphql.CollectFields(operation, child.Selections, nil) {
			switch item.Name {
			case "__typename", "hash", "mimeType", "size", "createdAt", "thumbnailURL", "reviewState", "exifStatus":
			default:
				return false
			}
		}
	}
	return true
}

func blossomObjectModel(row dbstore.BlossomObjectRow) *model.AdminBlossomObject {
	mediaPath := ""
	if config.Cfg != nil {
		mediaPath = config.Cfg.Store.MediaPath
	}
	directURL := func(hash, extension string) string {
		extension = strings.TrimSpace(extension)
		if extension != "" && !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		return strings.TrimRight(mediaPath, "/") + "/" + hash + extension
	}
	ext := strings.TrimPrefix(strings.TrimSpace(row.Extension), ".")
	if ext == "" {
		ext = "bin"
	}
	params := url.Values{}
	if parsed, err := url.Parse(strings.TrimSpace(mediaPath)); err == nil {
		hint := parsed.Host
		if hint == "" {
			hint = strings.Trim(parsed.Path, "/")
		}
		if hint != "" {
			params.Add("xs", hint)
		}
	}
	if strings.TrimSpace(row.UploaderPubkey) != "" {
		params.Add("as", row.UploaderPubkey)
	}
	if row.Size > 0 {
		params.Add("sz", strconv.FormatInt(row.Size, 10))
	}
	id := fmt.Sprintf("blossom:%s.%s", row.Hash, ext)
	if encoded := params.Encode(); encoded != "" {
		id += "?" + encoded
	}
	createdAt := ""
	if !row.CreatedAt.IsZero() {
		createdAt = row.CreatedAt.UTC().Format(time.RFC3339)
	}
	object := &model.AdminBlossomObject{
		Hash: row.Hash, UploaderPubkey: row.UploaderPubkey, MimeType: row.MIMEType, Extension: row.Extension,
		Size: int(row.Size), CreatedAt: createdAt, ReviewState: row.ReviewState, ExifStatus: row.ExifStatus,
		GpsDetected: row.GPSDetected, DirectURL: directURL(row.Hash, row.Extension), BlossomID: &id,
		DownloadCount: int(row.DownloadCount), IngressBytes: int(row.IngressBytes), EgressBytes: int(row.EgressBytes),
		FlagReason: optionalString(row.FlagReason), Mirrors: []string{}, Nip94Tags: []*model.NostrTag{},
	}
	if preview := internalblossom.BrowserURL(row.Hash); preview != "" {
		object.PreviewURL = &preview
	}
	if row.Width.Valid {
		object.Width = &row.Width.Int32
	}
	if row.Height.Valid {
		object.Height = &row.Height.Int32
	}
	if row.DurationMS.Valid {
		duration := int(row.DurationMS.Int64)
		object.DurationMs = &duration
	}
	if row.BitrateKbps.Valid {
		object.BitrateKbps = &row.BitrateKbps.Int32
	}
	if row.ThumbnailHash != "" {
		value := internalblossom.BrowserURL(row.ThumbnailHash)
		object.ThumbnailURL = &value
	}
	if row.OptimizedHash != "" {
		value := internalblossom.BrowserURL(row.OptimizedHash)
		object.OptimizedURL = &value
	}
	if err := json.Unmarshal(row.Mirrors, &object.Mirrors); err != nil || object.Mirrors == nil {
		object.Mirrors = []string{}
	}
	var tags [][]string
	if err := json.Unmarshal(row.NIP94Tags, &tags); err != nil {
		var legacy map[string]string
		if json.Unmarshal(row.NIP94Tags, &legacy) == nil {
			keys := make([]string, 0, len(legacy))
			for key := range legacy {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				tags = append(tags, []string{key, legacy[key]})
			}
		}
	}
	for _, tag := range tags {
		object.Nip94Tags = append(object.Nip94Tags, &model.NostrTag{Values: tag})
	}
	return object
}
