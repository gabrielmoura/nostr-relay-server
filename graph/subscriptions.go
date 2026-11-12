package graph

import (
	"context"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	"github.com/gabrielmoura/nostr-relay-server/internal/events"
)

const timeLayout = time.RFC3339

// SubscriptionsBroker adapts the shared event bus to the typed channels used
// by the gqlgen subscription resolvers. Every subscription owns its own
// channel; the bus is responsible for bounded fan-out and drop accounting.
type SubscriptionsBroker struct {
	bus *events.Bus
}

var defaultBroker = &SubscriptionsBroker{bus: events.Default()}

func GetSubscriptionsBroker() *SubscriptionsBroker { return defaultBroker }

func (b *SubscriptionsBroker) PublishPlanChange(plan *model.AdminBlossomPlan) {
	if plan == nil {
		return
	}
	b.bus.Publish(events.TopicBlossomPlan, plan)
}

func (b *SubscriptionsBroker) PublishObjectChange(object *model.AdminBlossomObject) {
	if object == nil {
		return
	}
	b.bus.Publish(events.TopicBlossomObject, object)
}

func (b *SubscriptionsBroker) SubscribePlanChanges(ctx context.Context) <-chan *model.AdminBlossomPlan {
	out := make(chan *model.AdminBlossomPlan, 16)
	source := b.bus.Subscribe(ctx, 32, events.TopicBlossomPlan)
	go func() {
		defer close(out)
		for event := range source {
			plan, err := events.Decode[model.AdminBlossomPlan](event)
			if err != nil {
				continue
			}
			select {
			case out <- &plan:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (b *SubscriptionsBroker) SubscribeObjectChanges(ctx context.Context) <-chan *model.AdminBlossomObject {
	out := make(chan *model.AdminBlossomObject, 16)
	source := b.bus.Subscribe(ctx, 32, events.TopicBlossomObject)
	go func() {
		defer close(out)
		for event := range source {
			object, err := events.Decode[model.AdminBlossomObject](event)
			if err != nil {
				continue
			}
			select {
			case out <- &object:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (b *SubscriptionsBroker) SubscribePresence(ctx context.Context) <-chan *model.AdminPresenceEvent {
	out := make(chan *model.AdminPresenceEvent, 16)
	source := b.bus.Subscribe(ctx, 32, events.TopicPresence)
	go func() {
		defer close(out)
		for event := range source {
			presence, err := events.Decode[events.PresenceEvent](event)
			if err != nil {
				continue
			}
			item := &model.AdminPresenceEvent{
				Action:            presence.Action,
				Wsid:              optionalString(presence.WSID),
				ActiveConnections: int32(presence.Active),
				AuthedConnections: int32(presence.Authed),
				ChangedAt:         presence.ChangedAt.UTC().Format(timeLayout),
			}
			select {
			case out <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// SubscribeJobs filters job events by queue, job id, and optional job names.
// Empty filters match everything.
func (b *SubscriptionsBroker) SubscribeJobs(ctx context.Context, queue, jobID string, names ...string) <-chan *model.AdminJobEvent {
	queue = strings.TrimSpace(queue)
	jobID = strings.TrimSpace(jobID)
	allowedNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name != "" {
			allowedNames[name] = struct{}{}
		}
	}

	out := make(chan *model.AdminJobEvent, 16)
	source := b.bus.Subscribe(ctx, 64, events.TopicJob)
	go func() {
		defer close(out)
		for event := range source {
			job, err := events.Decode[events.JobEvent](event)
			if err != nil {
				continue
			}
			if queue != "" && job.Queue != queue {
				continue
			}
			if jobID != "" && job.ID != jobID {
				continue
			}
			if len(allowedNames) > 0 {
				if _, ok := allowedNames[job.Name]; !ok {
					continue
				}
			}
			item := &model.AdminJobEvent{
				ID:              job.ID,
				Queue:           job.Queue,
				Name:            job.Name,
				Status:          job.Status,
				Attempts:        int32(job.Attempts),
				MaxAttempts:     int32(job.MaxAttempts),
				ProgressCurrent: int(job.ProgressCurrent),
				ProgressTotal:   int(job.ProgressTotal),
				Stage:           optionalString(job.Stage),
				Error:           optionalString(job.Error),
				UpdatedAt:       job.UpdatedAt.UTC().Format(timeLayout),
			}
			select {
			case out <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
