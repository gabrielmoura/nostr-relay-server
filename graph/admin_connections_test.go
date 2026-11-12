package graph

import (
	"bytes"
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
)

func TestConnectionQueriesGraphQLUseListenerSnapshotsWithPagination(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		active      []listener.ConnectionInfo
		authed      []listener.ConnectionInfo
		wantWSID    string
		wantOffset  int
		wantItem    connectionQueryItem
		wantHasMore bool
		wantActive  int
		wantAuthed  int
	}{
		{
			name:  "active connections",
			query: `query { activeConnections(page: {limit: 1, offset: 1}) { pageInfo { total limit offset hasMore } items { wsid ip authed subscriptionCount connectedAt lastSeenAt userAgent } } }`,
			active: []listener.ConnectionInfo{
				{WSID: "ws-first", IP: "192.0.2.1", SubscriptionCount: 1, ConnectedAt: 1, LastSeenAt: 2},
				{WSID: "ws-second", IP: "192.0.2.2", Authed: "pubkey", SubscriptionCount: 3, ConnectedAt: 10, LastSeenAt: 20, UserAgent: "nostr-client"},
			},
			wantWSID:    "ws-second",
			wantOffset:  1,
			wantItem:    connectionQueryItem{WSID: "ws-second", IP: "192.0.2.2", Authed: "pubkey", SubscriptionCount: 3, ConnectedAt: "1970-01-01T00:00:10Z", LastSeenAt: "1970-01-01T00:00:20Z", UserAgent: "nostr-client"},
			wantHasMore: false,
			wantActive:  1,
		},
		{
			name:  "authenticated connections",
			query: `query { authedConnections(page: {limit: 1, offset: 0}) { pageInfo { total limit offset hasMore } items { wsid ip authed subscriptionCount connectedAt lastSeenAt userAgent } } }`,
			authed: []listener.ConnectionInfo{
				{WSID: "ws-auth-first", IP: "198.51.100.1", Authed: "pubkey-first", SubscriptionCount: 2, ConnectedAt: 30, LastSeenAt: 40, UserAgent: "client-first"},
				{WSID: "ws-auth-second", IP: "198.51.100.2", Authed: "pubkey-second", SubscriptionCount: 4, ConnectedAt: 50, LastSeenAt: 60, UserAgent: "client-second"},
			},
			wantWSID:    "ws-auth-first",
			wantOffset:  0,
			wantItem:    connectionQueryItem{WSID: "ws-auth-first", IP: "198.51.100.1", Authed: "pubkey-first", SubscriptionCount: 2, ConnectedAt: "1970-01-01T00:00:30Z", LastSeenAt: "1970-01-01T00:00:40Z", UserAgent: "client-first"},
			wantHasMore: true,
			wantAuthed:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activeCalls, authedCalls := 0, 0
			previousActive := activeConnectionsSnapshot
			previousAuthed := authedConnectionsSnapshot
			activeConnectionsSnapshot = func() []listener.ConnectionInfo {
				activeCalls++
				return tt.active
			}
			authedConnectionsSnapshot = func() []listener.ConnectionInfo {
				authedCalls++
				return tt.authed
			}
			t.Cleanup(func() {
				activeConnectionsSnapshot = previousActive
				authedConnectionsSnapshot = previousAuthed
			})

			req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":`+stdjsonQuote(tt.query)+`}`))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			httpHandler(&Resolver{}).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}

			var response connectionQueryResponse
			if err := stdjson.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode GraphQL response: %v", err)
			}
			if len(response.Errors) != 0 {
				t.Fatalf("GraphQL errors = %#v", response.Errors)
			}

			page := response.Data.ActiveConnections
			if page == nil {
				page = response.Data.AuthedConnections
			}
			if page == nil {
				t.Fatal("connection page is missing")
			}
			if page.PageInfo.Total != 2 || page.PageInfo.Limit != 1 || page.PageInfo.Offset != tt.wantOffset || page.PageInfo.HasMore != tt.wantHasMore {
				t.Fatalf("pageInfo = %#v", page.PageInfo)
			}
			if len(page.Items) != 1 || page.Items[0] != tt.wantItem || page.Items[0].WSID != tt.wantWSID {
				t.Fatalf("items = %#v", page.Items)
			}
			if tt.wantActive != activeCalls || tt.wantAuthed != authedCalls {
				t.Fatalf("snapshot calls active=%d authed=%d, want active=%d authed=%d", activeCalls, authedCalls, tt.wantActive, tt.wantAuthed)
			}
		})
	}
}

func stdjsonQuote(value string) string {
	encoded, _ := stdjson.Marshal(value)
	return string(encoded)
}

type connectionQueryResponse struct {
	Data struct {
		ActiveConnections *connectionQueryPage `json:"activeConnections"`
		AuthedConnections *connectionQueryPage `json:"authedConnections"`
	} `json:"data"`
	Errors []any `json:"errors"`
}

type connectionQueryPage struct {
	PageInfo connectionQueryPageInfo `json:"pageInfo"`
	Items    []connectionQueryItem   `json:"items"`
}

type connectionQueryPageInfo struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"hasMore"`
}

type connectionQueryItem struct {
	WSID              string `json:"wsid"`
	IP                string `json:"ip"`
	Authed            string `json:"authed"`
	SubscriptionCount int    `json:"subscriptionCount"`
	ConnectedAt       string `json:"connectedAt"`
	LastSeenAt        string `json:"lastSeenAt"`
	UserAgent         string `json:"userAgent"`
}
