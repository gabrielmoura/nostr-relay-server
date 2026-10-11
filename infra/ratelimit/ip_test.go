package ratelimit

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestPerIPUsesIndependentTokenBuckets(t *testing.T) {
	limiter := New(Config{Enabled: true, RequestsPerSec: rate.Limit(1), Burst: 1, MaxClients: 10, IdleTTL: time.Minute})
	now := time.Unix(0, 0)
	limiter.now = func() time.Time { return now }

	if !limiter.Allow("192.0.2.1") || limiter.Allow("192.0.2.1") {
		t.Fatal("first IP should be limited after its burst")
	}
	if !limiter.Allow("192.0.2.2") {
		t.Fatal("second IP must have an independent bucket")
	}
}

func TestPerIPCanBeDisabled(t *testing.T) {
	limiter := New(Config{Enabled: false})
	for range 3 {
		if !limiter.Allow("192.0.2.1") {
			t.Fatal("disabled limiter rejected a request")
		}
	}
}

func TestPerIPPrunesIdleClientsBeforeApplyingCapacity(t *testing.T) {
	limiter := New(Config{Enabled: true, RequestsPerSec: rate.Limit(10), Burst: 1, MaxClients: 1, IdleTTL: time.Minute})
	now := time.Unix(0, 0)
	limiter.now = func() time.Time { return now }
	if !limiter.Allow("192.0.2.1") || limiter.Allow("192.0.2.2") {
		t.Fatal("capacity must reject a new active client")
	}
	now = now.Add(time.Minute)
	if !limiter.Allow("192.0.2.2") {
		t.Fatal("expired client should be pruned before capacity is checked")
	}
}
