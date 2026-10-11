package net

import (
	"bytes"
	"io"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	assets "github.com/gabrielmoura/nostr-relay-server/internal/embed"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestSetupExternalRoutesServesEmbeddedNostrIcon(t *testing.T) {
	app := fiber.New()
	factory := &RouterFactory{Config: &config.Config{}}
	factory.setupExternalRoutes(app)

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/nostr.png", nil))
	if err != nil {
		t.Fatalf("icon request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if got := response.Header.Get(fiber.HeaderContentType); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read icon response: %v", err)
	}
	if !bytes.Equal(body, assets.NostrPNG) {
		t.Fatal("icon response does not match embedded asset")
	}
}

func TestFiberConfigTrustedProxies(t *testing.T) {
	tests := []struct {
		name             string
		trustedProxies   []string
		wantProxyHeader  string
		wantIPValidation bool
		wantTrustedCheck bool
	}{
		{
			name: "direct peers ignore forwarded headers",
		},
		{
			name:             "configured proxies enable forwarded headers",
			trustedProxies:   []string{"10.0.0.0/8", "2001:db8::/32"},
			wantProxyHeader:  fiber.HeaderXForwardedFor,
			wantIPValidation: true,
			wantTrustedCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &RouterFactory{
				Config: &config.Config{
					Security: config.SecurityConfig{TrustedProxies: tt.trustedProxies},
				},
			}

			fiberConfig := factory.fiberConfig()
			if fiberConfig.ProxyHeader != tt.wantProxyHeader {
				t.Fatalf("ProxyHeader = %q, want %q", fiberConfig.ProxyHeader, tt.wantProxyHeader)
			}
			if fiberConfig.EnableIPValidation != tt.wantIPValidation {
				t.Fatalf("EnableIPValidation = %t, want %t", fiberConfig.EnableIPValidation, tt.wantIPValidation)
			}
			if fiberConfig.EnableTrustedProxyCheck != tt.wantTrustedCheck {
				t.Fatalf("EnableTrustedProxyCheck = %t, want %t", fiberConfig.EnableTrustedProxyCheck, tt.wantTrustedCheck)
			}
			if len(fiberConfig.TrustedProxies) != len(tt.trustedProxies) {
				t.Fatalf("TrustedProxies = %v, want %v", fiberConfig.TrustedProxies, tt.trustedProxies)
			}
		})
	}
}

func TestFiberConfigClientIPTrustsOnlyConfiguredProxies(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies []string
		remoteIP       string
		forwardedFor   string
		wantIP         string
	}{
		{
			name:           "untrusted peer cannot spoof forwarded IP",
			trustedProxies: []string{"10.0.0.10"},
			remoteIP:       "198.51.100.10",
			forwardedFor:   "203.0.113.5",
			wantIP:         "198.51.100.10",
		},
		{
			name:           "exact trusted proxy supplies client IP",
			trustedProxies: []string{"10.0.0.10"},
			remoteIP:       "10.0.0.10",
			forwardedFor:   "203.0.113.5",
			wantIP:         "203.0.113.5",
		},
		{
			name:           "trusted proxy CIDR supplies first valid client IP",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteIP:       "10.1.2.3",
			forwardedFor:   "invalid, 203.0.113.5, 198.51.100.4",
			wantIP:         "203.0.113.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &RouterFactory{
				Config: &config.Config{
					Security: config.SecurityConfig{TrustedProxies: tt.trustedProxies},
				},
			}
			app := fiber.New(factory.fiberConfig())
			fastCtx := &fasthttp.RequestCtx{}
			fastCtx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(tt.remoteIP), Port: 9090})
			fastCtx.Request.Header.Set(fiber.HeaderXForwardedFor, tt.forwardedFor)

			ctx := app.AcquireCtx(fastCtx)
			defer app.ReleaseCtx(ctx)

			if got := ctx.IP(); got != tt.wantIP {
				t.Fatalf("IP() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}
