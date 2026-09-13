package graph

import (
	"context"
	"strings"
	"testing"
)

func TestGraphQLRecoverPresentsInternalErrorWithoutTechnicalDetail(t *testing.T) {
	t.Setenv("NRSERVER_GRAPHQL_DEBUG", "false")

	presented := graphQLErrorPresenter(
		context.Background(),
		graphQLRecover(context.Background(), "database password leaked"),
	)

	if presented.Message != "Ocorreu um erro interno ao processar a solicitação." {
		t.Fatalf("message = %q, want friendly internal error", presented.Message)
	}
	if presented.Extensions["code"] != GraphQLErrorCodeInternalError {
		t.Fatalf("code = %#v, want %q", presented.Extensions["code"], GraphQLErrorCodeInternalError)
	}
	if _, ok := presented.Extensions["detail"]; ok {
		t.Fatalf("extensions = %#v, want no technical detail outside debug mode", presented.Extensions)
	}
}

func TestGraphQLErrorPresenterIncludesTechnicalDetailInDebugMode(t *testing.T) {
	t.Setenv("NRSERVER_GRAPHQL_DEBUG", "true")

	presented := graphQLErrorPresenter(
		context.Background(),
		graphQLRecover(context.Background(), "database password leaked"),
	)
	detail, ok := presented.Extensions["detail"].(string)
	if !ok || !strings.Contains(detail, "database password leaked") {
		t.Fatalf("detail = %#v, want recovered panic detail", presented.Extensions["detail"])
	}
}
