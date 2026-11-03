package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
)

type fakeBlossomObjectSource struct {
	filter        dbstore.BlossomObjectFilters
	limit, offset int
}

func (source *fakeBlossomObjectSource) ListBlossomObjects(_ context.Context, filter dbstore.BlossomObjectFilters, limit, offset int) ([]dbstore.BlossomObjectRow, int64, error) {
	source.filter, source.limit, source.offset = filter, limit, offset
	return []dbstore.BlossomObjectRow{{Hash: "abc", MIMEType: "image/png", ReviewState: "ready", ExifStatus: "pending"}}, 9, nil
}

func TestBlossomObjectsGraphQLPassesSearchAndPaginationToSource(t *testing.T) {
	source := &fakeBlossomObjectSource{}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"query ($filter: AdminBlossomObjectFilterInput, $page: OffsetPageInput) { blossomObjects(filter: $filter, page: $page) { pageInfo { total hasMore } items { hash mimeType } } }", "variables":{"filter":{"q":"  image  "},"page":{"limit":2,"offset":3}}}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	httpHandler(&Resolver{blossomObjectSource: source}).ServeHTTP(recorder, req)
	var response struct {
		Data struct {
			BlossomObjects struct {
				PageInfo struct {
					Total   int  `json:"total"`
					HasMore bool `json:"hasMore"`
				} `json:"pageInfo"`
				Items []struct {
					Hash     string `json:"hash"`
					MIMEType string `json:"mimeType"`
				} `json:"items"`
			} `json:"blossomObjects"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || len(response.Errors) != 0 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if source.filter.Query != "image" || source.limit != 2 || source.offset != 3 {
		t.Fatalf("source received filter=%+v limit=%d offset=%d", source.filter, source.limit, source.offset)
	}
	if len(response.Data.BlossomObjects.Items) != 1 || response.Data.BlossomObjects.Items[0].Hash != "abc" || response.Data.BlossomObjects.PageInfo.Total != 9 || !response.Data.BlossomObjects.PageInfo.HasMore {
		t.Fatalf("unexpected data: %+v", response.Data.BlossomObjects)
	}
}

func TestBlossomObjectModelPreservesRepeatedTags(t *testing.T) {
	object := blossomObjectModel(dbstore.BlossomObjectRow{
		Hash: "abc", Extension: "png", NIP94Tags: []byte(`[["t","one"],["t","two"]]`),
		Mirrors: []byte(`["https://example.com"]`),
	})
	if len(object.Nip94Tags) != 2 || object.Nip94Tags[0].Values[1] != "one" || object.Nip94Tags[1].Values[1] != "two" {
		t.Fatalf("tags = %#v", object.Nip94Tags)
	}
	if len(object.Mirrors) != 1 || object.Mirrors[0] != "https://example.com" {
		t.Fatalf("mirrors = %#v", object.Mirrors)
	}
}
