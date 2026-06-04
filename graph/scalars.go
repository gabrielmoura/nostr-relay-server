package graph

import (
	"time"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalTime(value time.Time) graphql.Marshaler {
	return graphql.MarshalTime(value)
}

func UnmarshalTime(value any) (time.Time, error) {
	return graphql.UnmarshalTime(value)
}
