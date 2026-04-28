package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type Manager struct {
	enabled       bool
	queries       *dbstore.Queries
	cfg           config.NIP29Config
	relayScope    string
	relayPubKey   string
	relayPrivKey  string
	roleConfigs   map[string]config.NIP29RoleConfig
	roleIDs       map[string]int32
	memberRoleID  int32
	creatorRoleID int32
}

type EventQueryFunc func(context.Context, nostr.Filter) (chan *nostr.Event, error)

var M *Manager

func Init(queries *dbstore.Queries) error {
	if config.Cfg == nil || !config.Cfg.NIP29.Enabled {
		M = &Manager{enabled: false}
		return nil
	}

	privKey := strings.TrimSpace(config.Cfg.RelayInformation.PrivKey)
	if privKey == "" {
		return fmt.Errorf("nip29 enabled but relay_information.priv_key is empty")
	}

	pubKey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		return fmt.Errorf("deriving relay pubkey for nip29: %w", err)
	}

	mgr := &Manager{
		enabled:      true,
		queries:      queries,
		cfg:          config.Cfg.NIP29,
		relayScope:   resolveRelayScope(config.Cfg),
		relayPubKey:  pubKey,
		relayPrivKey: privKey,
		roleConfigs:  make(map[string]config.NIP29RoleConfig),
		roleIDs:      make(map[string]int32),
	}

	if err := mgr.bootstrapRoles(context.Background()); err != nil {
		return err
	}
	if err := mgr.refreshActiveGroupsMetric(context.Background()); err != nil {
		log.Logger.Warn("failed to refresh nip29 active groups metric", zap.Error(err))
	}

	M = mgr
	return nil
}

func (m *Manager) GetRelayScope() string {
	if m == nil {
		return ""
	}
	return m.relayScope
}

func Enabled() bool {
	return M != nil && M.enabled
}

func QueryEvents(ctx context.Context, authed string, filter nostr.Filter, upstream EventQueryFunc) (chan *nostr.Event, bool, error) {
	if !Enabled() || !M.shouldHandleFilter(filter) {
		return nil, false, nil
	}

	start := time.Now()
	defer func() {
		metrics.NostrNIP29ProcessingSeconds.WithLabelValues("query_events").Observe(time.Since(start).Seconds())
	}()

	if reject, reason := M.validateFilter(ctx, authed, filter); reject {
		return nil, true, errors.New(reason)
	}

	results, err := upstream(ctx, filter)
	if err != nil {
		return nil, true, err
	}

	out := make(chan *nostr.Event)
	go M.forwardAllowedEvents(ctx, authed, results, out)
	return out, true, nil
}

func CountEvents(ctx context.Context, authed string, filter nostr.Filter, upstream EventQueryFunc) (int64, bool, error) {
	ch, handled, err := QueryEvents(ctx, authed, filter, upstream)
	if !handled || err != nil {
		return 0, handled, err
	}

	var total int64
	for range ch {
		total++
	}
	return total, true, nil
}

func ValidateFilter(ctx context.Context, authed string, filter nostr.Filter) (bool, string) {
	if !Enabled() {
		return false, ""
	}
	return M.validateFilter(ctx, authed, filter)
}

func ValidateIncomingEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	if !Enabled() || evt == nil || !M.isRelevantEvent(evt) {
		return false, ""
	}

	metrics.NostrNIP29EventsReceivedTotal.WithLabelValues(metrics.GetKindName(evt.Kind)).Inc()
	start := time.Now()
	defer func() {
		metrics.NostrNIP29ProcessingSeconds.WithLabelValues("validate_event").Observe(time.Since(start).Seconds())
	}()

	return M.validateIncomingEvent(ctx, evt)
}

func AfterStoreEvent(ctx context.Context, evt *nostr.Event) error {
	if !Enabled() || evt == nil || !M.isRelevantEvent(evt) {
		return nil
	}

	start := time.Now()
	defer func() {
		metrics.NostrNIP29ProcessingSeconds.WithLabelValues("after_store").Observe(time.Since(start).Seconds())
	}()

	return M.afterStoreEvent(ctx, evt)
}

func resolveRelayScope(cfg *config.Config) string {
	scope := strings.TrimSpace(cfg.NIP29.RelayScope)
	if scope != "" {
		return scope
	}
	if strings.TrimSpace(cfg.RelayInformation.CanonicalURL) != "" {
		return strings.TrimSpace(cfg.RelayInformation.CanonicalURL)
	}
	return strings.TrimSpace(cfg.RelayInformation.URL)
}
