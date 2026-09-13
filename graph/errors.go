package graph

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	// GraphQLErrorCodeBadInput identifies malformed GraphQL arguments or input.
	GraphQLErrorCodeBadInput = "BAD_INPUT"
	// GraphQLErrorCodeInternalError identifies an unexpected server failure.
	GraphQLErrorCodeInternalError = "INTERNAL_ERROR"
	// GraphQLErrorCodeNotFound identifies a requested resource that does not exist.
	GraphQLErrorCodeNotFound = "NOT_FOUND"
	// GraphQLErrorCodeUnauthorized identifies a request without sufficient credentials.
	GraphQLErrorCodeUnauthorized = "UNAUTHORIZED"
)

const startNegentropySyncErrorMessage = "Não foi possível iniciar a sincronização. Verifique os filtros informados."

type clientGraphQLError struct {
	code    string
	message string
	err     error
}

func (e *clientGraphQLError) Error() string {
	return e.message
}

func (e *clientGraphQLError) Unwrap() error {
	return e.err
}

func newClientGraphQLError(code string, message string, err error) error {
	return &clientGraphQLError{
		code:    code,
		message: message,
		err:     err,
	}
}

func graphQLErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	presented := graphql.DefaultErrorPresenter(ctx, err)
	code := GraphQLErrorCodeInternalError
	message := "Não foi possível processar a solicitação."

	var clientErr *clientGraphQLError
	if errors.As(err, &clientErr) {
		code = clientErr.code
		message = clientErr.message
	} else if strings.HasPrefix(presented.Path.String(), "startNegentropySync.input") {
		code = GraphQLErrorCodeBadInput
		message = startNegentropySyncErrorMessage
	} else {
		var gqlErr *gqlerror.Error
		if errors.As(err, &gqlErr) && gqlErr.Rule != "" {
			code = GraphQLErrorCodeBadInput
			message = "A solicitação GraphQL é inválida."
		}
	}

	presented.Message = message
	presented.Extensions = map[string]any{"code": code}
	if graphQLDebugEnabled() {
		presented.Extensions["detail"] = graphQLErrorDetail(err)
	}

	return presented
}

func graphQLErrorDetail(err error) string {
	var clientErr *clientGraphQLError
	if errors.As(err, &clientErr) && clientErr.err != nil {
		return clientErr.err.Error()
	}
	return err.Error()
}

func graphQLRecover(_ context.Context, recovered any) error {
	return newClientGraphQLError(
		GraphQLErrorCodeInternalError,
		"Ocorreu um erro interno ao processar a solicitação.",
		fmt.Errorf("panic recovered: %v", recovered),
	)
}

func graphQLDebugEnabled() bool {
	debug, err := strconv.ParseBool(os.Getenv("NRSERVER_GRAPHQL_DEBUG"))
	return err == nil && debug
}
