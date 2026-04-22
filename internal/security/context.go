package security

import "context"

type bypassContextKey struct{}

type BypassContext struct {
	IPWhitelisted     bool
	PubKeyWhitelisted bool
}

func WithBypassContext(ctx context.Context, bypass BypassContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, bypassContextKey{}, bypass)
}

func BypassFromContext(ctx context.Context) BypassContext {
	if ctx == nil {
		return BypassContext{}
	}
	bypass, ok := ctx.Value(bypassContextKey{}).(BypassContext)
	if !ok {
		return BypassContext{}
	}
	return bypass
}

func (b BypassContext) PublicationRestrictionsBypassed() bool {
	return b.IPWhitelisted || b.PubKeyWhitelisted
}

func (b BypassContext) RequestRestrictionsBypassed() bool {
	return b.IPWhitelisted
}

func (b BypassContext) RateLimitsBypassed() bool {
	return b.IPWhitelisted || b.PubKeyWhitelisted
}
