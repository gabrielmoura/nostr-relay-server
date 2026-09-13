package graph

import (
	"bytes"
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGraphQLResponseIncludesRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(requestIDHeader, "request-success-test")
	recorder := httptest.NewRecorder()
	HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Header().Get(requestIDHeader) != "request-success-test" {
		t.Fatalf("%s response header = %q, want request ID", requestIDHeader, recorder.Header().Get(requestIDHeader))
	}

	var response struct {
		Extensions map[string]any `json:"extensions"`
	}
	if err := stdjson.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GraphQL response: %v", err)
	}
	if response.Extensions["requestId"] != "request-success-test" {
		t.Fatalf("response extensions = %#v, want request ID", response.Extensions)
	}
}
