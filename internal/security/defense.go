package security

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	redisinfra "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

const counterScriptSrc = "local count = redis.call('INCR', KEYS[1]); if count == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]); end; return count"

type defenseDecision struct {
	prefix Prefix
	msg    string
	action string
}

type defender struct {
	enabled  bool
	useRedis bool
	blockTTL time.Duration
	client   *redisinfra.Client
	script   *goredis.Script
	event    config.SecurityWindowLimitConfig
	req      config.SecurityWindowLimitConfig
}

func newDefender(cfg config.SecurityDefenseConfig, client *redisinfra.Client) *defender {
	return &defender{
		enabled:  cfg.Enabled,
		useRedis: cfg.UseRedis,
		blockTTL: time.Duration(cfg.BlockTTLSeconds) * time.Second,
		client:   client,
		script:   goredis.NewScript(counterScriptSrc),
		event:    cfg.Event,
		req:      cfg.Req,
	}
}

func (d *defender) evaluateEvent(ctx context.Context, ip, pubkey string) (defenseDecision, error) {
	return d.evaluate(ctx, "event", d.event, ip, pubkey)
}

func (d *defender) evaluateReq(ctx context.Context, ip, pubkey string) (defenseDecision, error) {
	return d.evaluate(ctx, "req", d.req, ip, pubkey)
}

func (d *defender) evaluate(ctx context.Context, scope string, cfg config.SecurityWindowLimitConfig, ip, pubkey string) (defenseDecision, error) {
	if !d.enabled || !d.useRedis || d.client == nil || !d.client.IsEnabled() || cfg.WindowSeconds <= 0 {
		return defenseDecision{}, nil
	}

	for _, subject := range []struct {
		name  string
		value string
	}{
		{name: "ip", value: ip},
		{name: "pubkey", value: pubkey},
	} {
		if subject.value == "" {
			continue
		}
		decision, err := d.evaluateSubject(ctx, scope, subject.name, subject.value, cfg)
		if err != nil || decision.action != "" {
			return decision, err
		}
	}

	return defenseDecision{}, nil
}

func (d *defender) evaluateSubject(ctx context.Context, scope, subject, value string, cfg config.SecurityWindowLimitConfig) (defenseDecision, error) {
	blockedKey := fmt.Sprintf("security:defense:%s:%s:%s:block", scope, subject, value)
	if ttl, err := d.client.TTL(ctx, blockedKey); err == nil && ttl > 0 {
		return d.record(scope, "blocked", PrefixBlocked, "temporary block in effect"), nil
	}

	countKey := fmt.Sprintf("security:defense:%s:%s:%s:count", scope, subject, value)
	count, err := d.script.Run(ctx, d.client.Raw(), []string{countKey}, cfg.WindowSeconds).Int64()
	if err != nil {
		return defenseDecision{}, err
	}

	if cfg.TemporaryBlock > 0 && count >= int64(cfg.TemporaryBlock) {
		if err := d.client.Set(ctx, blockedKey, "1", d.blockTTL); err != nil {
			return defenseDecision{}, err
		}
		return d.record(scope, "blocked", PrefixBlocked, "temporary block applied"), nil
	}
	if cfg.RestrictAfter > 0 && count >= int64(cfg.RestrictAfter) {
		return d.record(scope, "restricted", PrefixRestricted, "temporary restrictions applied"), nil
	}
	if cfg.ThrottleAfter > 0 && count >= int64(cfg.ThrottleAfter) {
		return d.record(scope, "throttled", PrefixRateLimited, "slow down there chief"), nil
	}

	return defenseDecision{}, nil
}

func (d *defender) record(scope, action string, prefix Prefix, message string) defenseDecision {
	metrics.NostrSecurityDefenseActionsTotal.WithLabelValues(scope, action).Inc()
	return defenseDecision{prefix: prefix, msg: message, action: action}
}
