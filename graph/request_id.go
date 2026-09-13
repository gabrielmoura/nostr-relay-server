package graph

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := requestIDFromHeader(r.Header)
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID)))
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func requestIDFromHeader(header http.Header) string {
	requestID := strings.TrimSpace(header.Get(requestIDHeader))
	if len(requestID) > 128 {
		return ""
	}
	return requestID
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(bytes)
}

func withRequestIDResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	response := next(ctx)
	if response == nil {
		return nil
	}

	if response.Extensions == nil {
		response.Extensions = map[string]any{}
	}
	response.Extensions["requestId"] = requestIDFromContext(ctx)
	return response
}
