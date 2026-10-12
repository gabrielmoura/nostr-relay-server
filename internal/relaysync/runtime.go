package relaysync

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncfiber "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/fiber"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

const effectiveIPHeader = "X-RelaySync-Effective-IP"

type HTTPRuntime struct {
	Adapter  *core.HTTPAdapter
	Services relaysyncfiber.Services
	Sessions *core.SessionService
}

func Build(cfg config.RelaySyncConfig, queries *db.Queries) (*HTTPRuntime, error) {
	if !cfg.Enabled || !cfg.HTTPEnabled {
		return nil, nil
	}
	authorizer, err := NewAllowlistAuthorizer(cfg.AllowedIPs)
	if err != nil {
		return nil, err
	}
	snapshot, err := db.NewRelaySyncSnapshotProvider(queries, cfg.JournalEpoch, 30*time.Minute)
	if err != nil {
		return nil, err
	}
	sessions, err := core.NewSessionService(core.SessionServiceOptions{Store: core.NewMemorySessionStore(nil), Snapshots: snapshot, Authorizer: authorizer})
	if err != nil {
		return nil, err
	}
	manifest, err := core.NewManifestService(sessions, 64)
	if err != nil {
		return nil, err
	}
	reconcile, err := core.NewReconcileService(sessions, core.DefaultLimits())
	if err != nil {
		return nil, err
	}
	services := relaysyncfiber.Services{Sessions: sessions, Manifests: manifest, Reconcile: reconcile, Resolve: resolvePrincipal}
	if cfg.ApplyEnabled {
		writer, err := db.NewRelaySyncEventWriter(queries, core.DefaultLimits(), nil)
		if err != nil {
			_ = sessions.Close()
			return nil, err
		}
		transfer, err := core.NewTransferService(sessions, writer, core.DefaultLimits())
		if err != nil {
			_ = sessions.Close()
			return nil, err
		}
		services.Transfer = transfer
	}
	if cfg.LiveEnabled {
		journal, err := db.NewRelaySyncJournalProvider(queries, cfg.JournalEpoch)
		if err != nil {
			_ = sessions.Close()
			return nil, err
		}
		checkpoints := core.NewMemoryCheckpointStore()
		services.Live = &relaysyncfiber.LiveOptions{
			MaxFrameBytes: core.DefaultLimits().MaxFrameBytes,
			NewSync: func(ctx context.Context, peer string, hello *relaysyncv2.ClientHello) (*core.LiveSync, error) {
				scope := scopeFromProto(hello.GetScope())
				return core.NewLiveSync(journal, checkpoints, core.CheckpointKey{PeerID: peer, Scope: scope}, scope, core.LiveConfig{})
			},
		}
	}
	capabilities := &relaysyncv2.Capabilities{ProtocolVersions: []uint32{2}, Compressions: []relaysyncv2.Compression{relaysyncv2.Compression_COMPRESSION_NONE, relaysyncv2.Compression_COMPRESSION_ZSTD}, Limits: limitsProto(core.DefaultLimits()), Scopes: []string{"public"}, LiveSync: cfg.LiveEnabled, Implementation: "nrserver-relaysync"}
	adapter, err := core.NewHTTPAdapter(capabilities)
	if err != nil {
		_ = sessions.Close()
		return nil, err
	}
	return &HTTPRuntime{Adapter: adapter, Services: services, Sessions: sessions}, nil
}

func scopeFromProto(scope *relaysyncv2.Scope) core.Scope {
	result := core.Scope{}
	if scope == nil {
		return result
	}
	result.Name, result.Version = scope.Name, scope.Version
	copy(result.Hash[:], scope.Hash)
	return result
}

func limitsProto(l core.Limits) *relaysyncv2.Limits {
	return &relaysyncv2.Limits{MaxEventBytes: uint32(l.MaxEventBytes), MaxEventsPerBatch: uint32(l.MaxEventsPerBatch), MaxBatchBytes: uint32(l.MaxBatchBytes), MaxFrameBytes: uint32(l.MaxFrameBytes), MaxSessionTtlSeconds: uint32(l.MaxSessionTTL.Seconds()), MaxConcurrentRounds: uint32(l.MaxConcurrentRounds)}
}

func resolvePrincipal(r *http.Request) (core.Principal, bool) {
	value := strings.TrimSpace(r.Header.Get(effectiveIPHeader))
	if value == "" {
		value = r.RemoteAddr
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		}
	}
	if value == "" {
		return core.Principal{}, false
	}
	return core.Principal{ID: value}, true
}

func (r *HTTPRuntime) Close() error {
	if r == nil || r.Sessions == nil {
		return nil
	}
	return r.Sessions.Close()
}
