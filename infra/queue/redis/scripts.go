package redisqueue

import (
	"context"
	_ "embed"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

//go:embed lua/enqueue.lua
var enqueueLua string

//go:embed lua/start.lua
var startLua string

//go:embed lua/ack_success.lua
var ackSuccessLua string

//go:embed lua/retry.lua
var retryLua string

//go:embed lua/move_dead.lua
var moveDeadLua string

//go:embed lua/promote_delayed.lua
var promoteDelayedLua string

//go:embed lua/cancel.lua
var cancelLua string

//go:embed lua/defer.lua
var deferLua string

type Scripts struct {
	enqueue        *goredis.Script
	start          *goredis.Script
	ackSuccess     *goredis.Script
	retry          *goredis.Script
	moveDead       *goredis.Script
	promoteDelayed *goredis.Script
	cancel         *goredis.Script
	deferJob       *goredis.Script
}

func NewScripts() *Scripts {
	return &Scripts{
		enqueue:        goredis.NewScript(enqueueLua),
		start:          goredis.NewScript(startLua),
		ackSuccess:     goredis.NewScript(ackSuccessLua),
		retry:          goredis.NewScript(retryLua),
		moveDead:       goredis.NewScript(moveDeadLua),
		promoteDelayed: goredis.NewScript(promoteDelayedLua),
		cancel:         goredis.NewScript(cancelLua),
		deferJob:       goredis.NewScript(deferLua),
	}
}

func (s *Scripts) Load(ctx context.Context, client *goredis.Client) error {
	loaders := []struct {
		name   string
		script *goredis.Script
	}{
		{name: "enqueue", script: s.enqueue},
		{name: "start", script: s.start},
		{name: "ack_success", script: s.ackSuccess},
		{name: "retry", script: s.retry},
		{name: "move_dead", script: s.moveDead},
		{name: "promote_delayed", script: s.promoteDelayed},
		{name: "cancel", script: s.cancel},
		{name: "defer", script: s.deferJob},
	}

	for _, loader := range loaders {
		if err := loader.script.Load(ctx, client).Err(); err != nil {
			return fmt.Errorf("load %s lua script: %w", loader.name, err)
		}
	}

	return nil
}
