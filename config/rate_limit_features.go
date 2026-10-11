package config

import "fmt"

// ValidateRateLimitFeatures rejects enabled rate limiters that could otherwise
// be silently disabled by their runtime constructor.
func (cfg *Config) ValidateRateLimitFeatures() error {
	if cfg == nil {
		return nil
	}
	for name, limit := range map[string]IPRequestRateLimitConfig{
		"api.rate_limit":        cfg.API.RateLimit,
		"negentropy_rate_limit": cfg.NegentropyRateLimit,
	} {
		if !limit.Enabled {
			continue
		}
		if limit.RequestsPerSec <= 0 || limit.Burst <= 0 || limit.MaxClients <= 0 || limit.IdleTTLSeconds <= 0 {
			return fmt.Errorf("%s requires positive requests_per_second, burst, max_clients, and idle_ttl_seconds when enabled", name)
		}
	}
	return nil
}
