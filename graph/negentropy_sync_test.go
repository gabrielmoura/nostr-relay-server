package graph

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

func TestUnmarshalAdminNegentropySyncInputAcceptsFilterList(t *testing.T) {
	const author = "91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd"

	input, err := (&executionContext{}).unmarshalInputAdminNegentropySyncInput(
		context.Background(),
		map[string]any{
			"remote": "wss://relay.example",
			"filter": []any{
				map[string]any{"authors": []any{author}},
			},
		},
	)
	if err != nil {
		t.Fatalf("unmarshal negentropy sync input: %v", err)
	}

	if len(input.Filter) != 1 {
		t.Fatalf("filter count = %d, want 1", len(input.Filter))
	}
	authors, ok := input.Filter[0]["authors"].([]any)
	if !ok || len(authors) != 1 || authors[0] != author {
		t.Fatalf("filter authors = %#v, want [%q]", input.Filter[0]["authors"], author)
	}
}

func TestStartNegentropySyncAcceptsFilterList(t *testing.T) {
	const author = "91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd91bea5cd"

	previousConfig := config.Cfg
	previousJobs := jobcore.Default()
	config.Cfg = &config.Config{
		Jobs:  config.JobsConfig{Enabled: true},
		Redis: config.RedisConfig{Enabled: true, Queue: config.RedisQueueConfig{Enabled: true}},
	}
	jobcore.SetDefault(nil)
	t.Cleanup(func() {
		config.Cfg = previousConfig
		jobcore.SetDefault(previousJobs)
	})

	body, err := stdjson.Marshal(map[string]any{
		"query": `mutation StartNegentropySync($input: AdminNegentropySyncInput!) {
			startNegentropySync(input: $input) { status }
		}`,
		"variables": map[string]any{
			"input": map[string]any{
				"remote": "wss://relay.example",
				"filter": []any{
					map[string]any{"authors": []any{author}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal GraphQL request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	HTTPHandler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GraphQL response status = %d, want %d", recorder.Code, http.StatusOK)
	}

	response := recorder.Body.String()
	if !strings.Contains(response, startNegentropySyncErrorMessage) {
		t.Fatalf("GraphQL response = %s, want friendly error after input parsing", response)
	}
	if !strings.Contains(response, GraphQLErrorCodeInternalError) {
		t.Fatalf("GraphQL response = %s, want INTERNAL_ERROR code", response)
	}
	if strings.Contains(response, "sync queue runtime is not initialized") {
		t.Fatalf("GraphQL response leaked handler detail: %s", response)
	}
	if strings.Contains(response, "[]interface {} is not a map") {
		t.Fatalf("GraphQL response leaked type assertion error: %s", response)
	}
}

func TestStartNegentropySyncRejectsInvalidFilterWithBadInputError(t *testing.T) {
	body, err := stdjson.Marshal(map[string]any{
		"query": `mutation StartNegentropySync($input: AdminNegentropySyncInput!) {
			startNegentropySync(input: $input) { status }
		}`,
		"variables": map[string]any{
			"input": map[string]any{
				"remote": "wss://relay.example",
				"filter": []any{true},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal GraphQL request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	HTTPHandler().ServeHTTP(recorder, req)

	var response struct {
		Errors []struct {
			Message    string         `json:"message"`
			Extensions map[string]any `json:"extensions"`
		} `json:"errors"`
	}
	if err := stdjson.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GraphQL response: %v", err)
	}
	if len(response.Errors) != 1 {
		t.Fatalf("GraphQL errors = %#v, want one error", response.Errors)
	}

	graphqlError := response.Errors[0]
	if graphqlError.Message != startNegentropySyncErrorMessage {
		t.Fatalf("error message = %q, want %q", graphqlError.Message, startNegentropySyncErrorMessage)
	}
	if graphqlError.Extensions["code"] != GraphQLErrorCodeBadInput {
		t.Fatalf("error code = %#v, want %q", graphqlError.Extensions["code"], GraphQLErrorCodeBadInput)
	}
	if strings.Contains(graphqlError.Message, "is not a map") {
		t.Fatalf("GraphQL error leaked Go type assertion: %q", graphqlError.Message)
	}
}
