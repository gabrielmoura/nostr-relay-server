package graph

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"go.uber.org/zap"
)

func logNegentropySyncRequest(
	ctx context.Context,
	remote string,
	filterCount int,
	duration time.Duration,
	result string,
	code string,
) {
	if log.Logger == nil {
		return
	}

	log.Logger.Info(
		"admin graphql negentropy sync",
		zap.String("requestId", requestIDFromContext(ctx)),
		zap.String("operationName", graphQLOperationName(ctx)),
		zap.String("remote", remote),
		zap.Int("filterCount", filterCount),
		zap.Duration("duration", duration),
		zap.String("result", result),
		zap.String("code", code),
	)
}

func graphQLOperationName(ctx context.Context) string {
	if !graphql.HasOperationContext(ctx) {
		return "StartNegentropySync"
	}

	operationName := graphql.GetOperationContext(ctx).OperationName
	if operationName == "" {
		return "StartNegentropySync"
	}
	return operationName
}
