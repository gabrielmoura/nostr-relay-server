// Package ratelimit provides bounded, in-process per-IP token buckets.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Config defines one independently configurable per-IP limiter.
type Config struct {
	Enabled        bool
	RequestsPerSec rate.Limit
	Burst          int
	MaxClients     int
	IdleTTL        time.Duration
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// PerIP is safe for concurrent use. It does not start background goroutines;
// stale entries are removed while handling requests.
type PerIP struct {
	cfg     Config
	mu      sync.Mutex
	clients map[string]*client
	now     func() time.Time
}

func New(cfg Config) *PerIP {
	if cfg.RequestsPerSec <= 0 || cfg.Burst <= 0 || cfg.MaxClients <= 0 {
		cfg.Enabled = false
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}
	return &PerIP{cfg: cfg, clients: make(map[string]*client), now: time.Now}
}

// Allow consumes one token for ip. An empty key is refused while enabled so
// malformed connection metadata cannot share an unbounded bucket.
func (l *PerIP) Allow(ip string) bool {
	if l == nil || !l.cfg.Enabled {
		return true
	}
	if ip == "" {
		return false
	}

	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if existing := l.clients[ip]; existing != nil {
		existing.lastSeen = now
		return existing.limiter.AllowN(now, 1)
	}

	l.pruneExpired(now)
	if len(l.clients) >= l.cfg.MaxClients {
		return false
	}

	limiter := rate.NewLimiter(l.cfg.RequestsPerSec, l.cfg.Burst)
	l.clients[ip] = &client{limiter: limiter, lastSeen: now}
	return limiter.AllowN(now, 1)
}

func (l *PerIP) pruneExpired(now time.Time) {
	for ip, client := range l.clients {
		if now.Sub(client.lastSeen) >= l.cfg.IdleTTL {
			delete(l.clients, ip)
		}
	}
}
