package db

import (
	"context"
)

type Querier interface {
	DeleteEvent(ctx context.Context, id string) error
	DeleteOldsEvents(ctx context.Context, arg DeleteOldsEventsParams) error
}

var _ Querier = (*Queries)(nil)
