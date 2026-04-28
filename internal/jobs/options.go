package jobs

import "time"

type DispatchConfig struct {
	Queue       string
	Priority    Priority
	Timeout     time.Duration
	MaxAttempts uint8
	RunAt       time.Time
	UniqueKey   string
	UniqueFor   time.Duration
}

type DispatchOption func(*DispatchConfig)

func WithQueue(queue string) DispatchOption {
	return func(cfg *DispatchConfig) {
		cfg.Queue = queue
	}
}

func WithPriority(priority Priority) DispatchOption {
	return func(cfg *DispatchConfig) {
		cfg.Priority = priority.Normalize()
	}
}

func WithTimeout(timeout time.Duration) DispatchOption {
	return func(cfg *DispatchConfig) {
		cfg.Timeout = timeout
	}
}

func WithMaxAttempts(maxAttempts uint8) DispatchOption {
	return func(cfg *DispatchConfig) {
		cfg.MaxAttempts = maxAttempts
	}
}

func WithRunAt(runAt time.Time) DispatchOption {
	return func(cfg *DispatchConfig) {
		cfg.RunAt = runAt.UTC()
	}
}

func WithUnique(key string, ttl time.Duration) DispatchOption {
	return func(cfg *DispatchConfig) {
		cfg.UniqueKey = key
		cfg.UniqueFor = ttl
	}
}

type RetryPolicy struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Jitter    time.Duration
}
