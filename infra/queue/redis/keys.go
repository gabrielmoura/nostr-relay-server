package redisqueue

import (
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

type Keys struct {
	queue string
}

func NewKeys(queue string) Keys {
	return Keys{queue: queue}
}

func (k Keys) tag() string {
	return fmt.Sprintf("{%s}", k.queue)
}

func (k Keys) Queue() string {
	return k.queue
}

func (k Keys) Seq() string {
	return fmt.Sprintf("rq:%s:seq", k.tag())
}

func (k Keys) Stream(priority jobs.Priority) string {
	return fmt.Sprintf("rq:%s:stream:%s", k.tag(), priority.Normalize())
}

func (k Keys) Delayed() string {
	return fmt.Sprintf("rq:%s:delayed", k.tag())
}

func (k Keys) Dead() string {
	return fmt.Sprintf("rq:%s:dead", k.tag())
}

func (k Keys) Jobs() string {
	return fmt.Sprintf("rq:%s:jobs", k.tag())
}

func (k Keys) BodyPrefix() string {
	return fmt.Sprintf("rq:%s:body:", k.tag())
}

func (k Keys) Body(id jobs.JobID) string {
	return k.BodyPrefix() + id.String()
}

func (k Keys) ResultPrefix() string {
	return fmt.Sprintf("rq:%s:result:", k.tag())
}

func (k Keys) Result(id jobs.JobID) string {
	return k.ResultPrefix() + id.String()
}

func (k Keys) State() string {
	return fmt.Sprintf("rq:%s:state", k.tag())
}

func (k Keys) Attempts() string {
	return fmt.Sprintf("rq:%s:attempts", k.tag())
}

func (k Keys) MetaPrefix() string {
	return fmt.Sprintf("rq:%s:meta:", k.tag())
}

func (k Keys) Meta(id jobs.JobID) string {
	return k.MetaPrefix() + id.String()
}

func (k Keys) MetricsBucket(t time.Time) string {
	return fmt.Sprintf("rq:%s:metrics:%s", k.tag(), t.UTC().Format("200601021504"))
}

func (k Keys) WorkersBucket(t time.Time) string {
	return fmt.Sprintf("rq:%s:workers:%s", k.tag(), t.UTC().Format("200601021504"))
}

func (k Keys) Unique(fingerprint string) string {
	return fmt.Sprintf("rq:%s:unique:%s", k.tag(), fingerprint)
}
