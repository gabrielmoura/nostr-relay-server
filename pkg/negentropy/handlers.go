package negentropy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	infraCache "github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	negentropyv2 "github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2"
	negcachev2 "github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/cache"
	negcontractsv2 "github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/contracts"
	negmodelv2 "github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

var (
	builder = NewMessageBuilder()

	negManagerOnce sync.Once
	negManager     *negentropyv2.Manager
	negManagerErr  error

	negSessionsMu sync.Mutex
	negSessions   = make(map[string]struct{})
)

type relayEventStore struct{}

type metricsQueryCache struct {
	backend string
	inner   negcontractsv2.QueryCache
}

func (m metricsQueryCache) Get(key string) ([]negmodelv2.EventRef, bool) {
	refs, ok := m.inner.Get(key)
	result := "miss"
	if ok {
		result = "hit"
	}
	metrics.NostrNegentropyV2CacheTotal.WithLabelValues(m.backend, result).Inc()

	return refs, ok
}

func (m metricsQueryCache) Set(key string, refs []negmodelv2.EventRef, ttl time.Duration) {
	m.inner.Set(key, refs, ttl)
	metrics.NostrNegentropyV2CacheTotal.WithLabelValues(m.backend, "set").Inc()
}

func (m metricsQueryCache) Delete(key string) {
	m.inner.Delete(key)
	metrics.NostrNegentropyV2CacheTotal.WithLabelValues(m.backend, "delete").Inc()
}

func (m metricsQueryCache) PurgeExpired(now time.Time) {
	m.inner.PurgeExpired(now)
}

func (relayEventStore) QueryEventRefs(ctx context.Context, filter negmodelv2.Filter) ([]negmodelv2.EventRef, error) {
	nostrFilter := negFilterToNostr(filter)
	events, err := db.DbQueries.QueryEvents(ctx, nostrFilter)
	if err != nil {
		return nil, err
	}

	refs := make([]negmodelv2.EventRef, 0, len(events))
	for _, evt := range events {
		id, parseErr := negentropyv2.ParseEventIDHex(evt.ID)
		if parseErr != nil {
			continue
		}

		refs = append(refs, negmodelv2.EventRef{
			CreatedAt: uint64(evt.CreatedAt),
			ID:        id,
		})
	}

	return refs, nil
}

// HandleNegOpen inicia a sessão de reconciliação.
// Carrega os dados do DB uma única vez, cria o vetor e o cacheia.
func HandleNegOpen(ws *dto.WsServer, data dto.Data) error {
	result := "ok"
	defer func() {
		metrics.NostrNegentropyV2RequestsTotal.WithLabelValues("open", result).Inc()
	}()

	if len(data) < 3 {
		result = "error"
		return fmt.Errorf("invalid NEG-OPEN format")
	}

	// 1. Parse Inputs
	var subID string
	if err := json.Unmarshal(data[1], &subID); err != nil {
		result = "error"
		return fmt.Errorf("invalid subID: %w", err)
	}

	var filter nostr.Filter
	if err := json.Unmarshal(data[2], &filter); err != nil {
		result = "error"
		return fmt.Errorf("invalid filter: %w", err)
	}

	var payloadHex string
	if len(data) > 3 {
		_ = json.Unmarshal(data[3], &payloadHex)
	}

	mgr, err := getNegentropyManager()
	if err != nil {
		result = "error"
		return err
	}

	if filter.Limit == 0 || filter.Limit > 10000 {
		filter.Limit = 10000
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := mgr.Open(ctx, negentropyv2.OpenRequest{
		SessionID:         subID,
		Filter:            nostrFilterToNeg(filter),
		InitialMessageHex: payloadHex,
	})
	if err != nil {
		result = "error"
		return err
	}

	if resp.Type == negentropyv2.ResponseTypeError {
		result = "error"
		metrics.NostrNegentropyV2ProtocolErrorsTotal.Inc()
	} else {
		addActiveSession(subID)
	}

	ws.ChanSender <- responseEnvelope(resp)
	return nil
}

// HandleNegMsg processa as etapas seguintes da reconciliação.
// Usa o vetor em cache (RAM) para extrema velocidade.
func HandleNegMsg(ws *dto.WsServer, data dto.Data) error {
	result := "ok"
	defer func() {
		metrics.NostrNegentropyV2RequestsTotal.WithLabelValues("msg", result).Inc()
	}()

	if len(data) < 3 {
		result = "error"
		return fmt.Errorf("invalid NEG-MSG format")
	}

	var subID string
	if err := json.Unmarshal(data[1], &subID); err != nil {
		result = "error"
		return err
	}

	var payloadHex string
	if err := json.Unmarshal(data[2], &payloadHex); err != nil {
		result = "error"
		return err
	}

	mgr, err := getNegentropyManager()
	if err != nil {
		result = "error"
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := mgr.OnMessage(ctx, negentropyv2.MessageRequest{SessionID: subID, MessageHex: payloadHex})
	if err != nil {
		result = "error"
		return err
	}

	if resp.Type == negentropyv2.ResponseTypeError {
		result = "error"
		metrics.NostrNegentropyV2ProtocolErrorsTotal.Inc()
	}

	ws.ChanSender <- responseEnvelope(resp)
	if resp.Done {
		ws.ChanSender <- builder.Close(subID)
		mgr.Close(subID)
		removeActiveSession(subID)
	}

	return nil
}

func HandleNegClose(data dto.Data) error {
	result := "ok"
	defer func() {
		metrics.NostrNegentropyV2RequestsTotal.WithLabelValues("close", result).Inc()
	}()

	if len(data) < 2 {
		result = "error"
		return fmt.Errorf("invalid NEG-CLOSE format")
	}

	var subID string
	if err := json.Unmarshal(data[1], &subID); err != nil {
		result = "error"
		return err
	}

	mgr, err := getNegentropyManager()
	if err != nil {
		result = "error"
		return err
	}

	mgr.Close(subID)
	removeActiveSession(subID)

	return nil
}

// HandleNegNeed responde aos IDs solicitados pelo cliente.
func HandleNegNeed(ws *dto.WsServer, data dto.Data) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid NEG-NEED format")
	}

	var subID string
	json.Unmarshal(data[1], &subID)

	var needIDs []string
	if err := json.Unmarshal(data[2], &needIDs); err != nil {
		return fmt.Errorf("failed to parse need IDs: %w", err)
	}

	if len(needIDs) == 0 {
		return nil
	}

	// Busca apenas os IDs necessários
	haveEvents, err := db.DbQueries.QueryEvents(context.Background(), nostr.Filter{
		IDs: needIDs,
	})
	if err != nil {
		return err
	}

	haveBytes, err := json.Marshal(haveEvents)
	if err != nil {
		return err
	}

	ws.ChanSender <- builder.Have(subID, haveBytes)
	return nil
}

// HandleNegHave recebe eventos novos enviados pelo cliente.
func HandleNegHave(ws *dto.WsServer, data dto.Data) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid NEG-HAVE format")
	}

	var newEvents []*nostr.Event
	if err := json.Unmarshal(data[2], &newEvents); err != nil {
		return err
	}

	ctx := context.Background()
	savedCount := 0

	for _, event := range newEvents {
		err := db.DbQueries.InsertEvent(ctx, event)
		if err != nil {
			log.Logger.Debug("Skipping event import", zap.String("id", event.ID), zap.Error(err))
			continue
		}
		savedCount++
	}

	if savedCount > 0 {
		log.Logger.Info("Negentropy imported events", zap.Int("count", savedCount))
		metrics.NostrNegentropyV2EventsImportedTotal.Add(float64(savedCount))
	}

	return nil
}

func getNegentropyManager() (*negentropyv2.Manager, error) {
	negManagerOnce.Do(func() {
		if db.DbQueries == nil {
			negManagerErr = fmt.Errorf("database is not initialized")
			return
		}

		ttl := 30 * time.Second
		if config.Cfg != nil && config.Cfg.Redis.Cache.QueryTTL > 0 {
			ttl = time.Duration(config.Cfg.Redis.Cache.QueryTTL) * time.Second
		}

		opts := negentropyv2.Options{
			FrameSizeLimit: FrameSizeLimit,
			CacheTTL:       ttl,
			SessionTTL:     5 * time.Minute,
		}

		store := relayEventStore{}

		queryCache := negentropyv2.QueryCache(negentropyv2.NewMemoryQueryCache())
		cacheBackend := "memory"
		if redisClient := infraCache.GetRedis(); redisClient != nil && redisClient.IsEnabled() {
			queryCache = negcachev2.NewRedisQueryCache(redisClient, negcachev2.RedisOptions{
				Prefix:  "relay:negentropy:v2",
				Timeout: 150 * time.Millisecond,
			})
			cacheBackend = "redis"
		}

		queryCache = metricsQueryCache{backend: cacheBackend, inner: queryCache}

		negManager = negentropyv2.NewManager(store, queryCache, opts)
	})

	if negManagerErr != nil {
		return nil, negManagerErr
	}

	return negManager, nil
}

func responseEnvelope(resp negentropyv2.Response) []any {
	switch resp.Type {
	case negentropyv2.ResponseTypeError:
		return builder.Error(resp.SessionID, resp.Reason)
	case negentropyv2.ResponseTypeClosed:
		return builder.Close(resp.SessionID)
	default:
		return []any{MsgMsg, resp.SessionID, resp.MessageHex}
	}
}

func nostrFilterToNeg(filter nostr.Filter) negmodelv2.Filter {
	out := negmodelv2.Filter{
		IDs:     append([]string(nil), filter.IDs...),
		Authors: append([]string(nil), filter.Authors...),
		Kinds:   append([]int(nil), filter.Kinds...),
		Tags:    copyTagMap(filter.Tags),
		Search:  filter.Search,
	}

	if filter.Since != nil {
		since := uint64(*filter.Since)
		out.Since = &since
	}

	if filter.Until != nil {
		until := uint64(*filter.Until)
		out.Until = &until
	}

	if filter.Limit > 0 {
		limit := filter.Limit
		out.Limit = &limit
	}

	return out
}

func negFilterToNostr(filter negmodelv2.Filter) nostr.Filter {
	out := nostr.Filter{
		IDs:     append([]string(nil), filter.IDs...),
		Authors: append([]string(nil), filter.Authors...),
		Kinds:   append([]int(nil), filter.Kinds...),
		Tags:    copyTagMap(filter.Tags),
		Search:  filter.Search,
	}

	if filter.Since != nil {
		since := nostr.Timestamp(*filter.Since)
		out.Since = &since
	}

	if filter.Until != nil {
		until := nostr.Timestamp(*filter.Until)
		out.Until = &until
	}

	if filter.Limit != nil {
		out.Limit = *filter.Limit
	}

	return out
}

func copyTagMap(tags map[string][]string) map[string][]string {
	if len(tags) == 0 {
		return nil
	}

	out := make(map[string][]string, len(tags))
	for key, values := range tags {
		copied := append([]string(nil), values...)
		out[key] = copied
	}

	return out
}

func addActiveSession(subID string) {
	negSessionsMu.Lock()
	negSessions[subID] = struct{}{}
	metrics.NostrNegentropyV2SessionsActive.Set(float64(len(negSessions)))
	negSessionsMu.Unlock()
}

func removeActiveSession(subID string) {
	negSessionsMu.Lock()
	delete(negSessions, subID)
	metrics.NostrNegentropyV2SessionsActive.Set(float64(len(negSessions)))
	negSessionsMu.Unlock()
}
