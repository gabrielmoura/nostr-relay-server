package graph

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/graph/model"
)

func TestLoggedUsersGraphQLReturnsProfilesWithoutRESTBridge(t *testing.T) {
	limit := int32(50)
	source := &fakeLoggedUsersSource{page: &model.AdminLoggedUserPage{
		PageInfo: &model.PageInfo{Total: 2, Limit: limit},
		Items: []*model.AdminLoggedUser{
			{Profile: profileToLoggedUserProfile(profileRecord{pubkey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", displayName: "Alice"}, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), ConnectionCount: 2, ConnectionState: "stable"},
			{Profile: profileToLoggedUserProfile(profileRecord{}, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"), ConnectionCount: 1, ConnectionState: "attention"},
		},
	}}

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"query { loggedUsers(page: {limit: 50, offset: 0}) { pageInfo { total } items { profile { pubkey displayName } connectionCount connectionState } } }"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	httpHandler(&Resolver{loggedUsersSource: source}).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			LoggedUsers struct {
				Items []struct {
					Profile struct {
						Pubkey      string `json:"pubkey"`
						DisplayName string `json:"displayName"`
					} `json:"profile"`
				} `json:"items"`
			} `json:"loggedUsers"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	if err := stdjson.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GraphQL response: %v", err)
	}
	if len(response.Errors) != 0 {
		t.Fatalf("GraphQL errors = %#v", response.Errors)
	}
	if len(response.Data.LoggedUsers.Items) != 2 {
		t.Fatalf("items = %#v, want two users", response.Data.LoggedUsers.Items)
	}
	if got := response.Data.LoggedUsers.Items[1].Profile; got.Pubkey != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" || got.DisplayName != got.Pubkey {
		t.Fatalf("fallback profile = %#v, want pubkey fallback", got)
	}
	if !source.includeProfiles {
		t.Fatal("source did not receive profile selection")
	}
}

func TestLoggedUsersGraphQLSkipsProfileLookupWhenNotSelected(t *testing.T) {
	source := &fakeLoggedUsersSource{page: &model.AdminLoggedUserPage{
		PageInfo: &model.PageInfo{},
		Items:    []*model.AdminLoggedUser{{ConnectionCount: 1, ConnectionState: "attention"}},
	}}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"query { loggedUsers { items { connectionCount connectionState } } }"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	httpHandler(&Resolver{loggedUsersSource: source}).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || recorder.Body.String() == "" {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if source.includeProfiles {
		t.Fatal("source was asked for profiles when GraphQL selection omitted profile")
	}
}

type fakeLoggedUsersSource struct {
	page            *model.AdminLoggedUserPage
	includeProfiles bool
}

func (source *fakeLoggedUsersSource) LoggedUsers(_ context.Context, _ *model.OffsetPageInput, includeProfiles bool) (*model.AdminLoggedUserPage, error) {
	source.includeProfiles = includeProfiles
	return source.page, nil
}
