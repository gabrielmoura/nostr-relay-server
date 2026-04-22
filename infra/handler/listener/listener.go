package listener

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/pubsub"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type Listener struct {
	filters   nostr.Filters
	createdAt time.Time
}

type ListenerData struct {
	Filters   json.RawMessage `json:"filters"`
	CreatedAt int64           `json:"created_at"`
}

type ConnectionInfo struct {
	WSID              string `json:"ws_id"`
	IP                string `json:"ip"`
	Authed            string `json:"authed,omitempty"`
	SubscriptionCount int    `json:"subscription_count"`
	ConnectedAt       int64  `json:"connected_at"`
	LastSeenAt        int64  `json:"last_seen_at"`
	UserAgent         string `json:"user_agent,omitempty"`
}

var (
	localListeners      = make(map[*dto.WsServer]map[string]*Listener)
	localListenersMutex sync.RWMutex
	listenerCount       atomic.Int32
	wsToID              = make(map[*dto.WsServer]string)
	wsToIDMutex         sync.RWMutex
	cleanupOnce         sync.Once
)

const wsLastSeenPrefix = "ws:last_seen:"

func getWsID(ws *dto.WsServer) string {
	wsToIDMutex.RLock()
	defer wsToIDMutex.RUnlock()
	return wsToID[ws]
}

func setWsID(ws *dto.WsServer, id string) {
	wsToIDMutex.Lock()
	defer wsToIDMutex.Unlock()
	wsToID[ws] = id
}

func Init() {
	if pubsub.GetPubSub() != nil && pubsub.GetPubSub().IsEnabled() {
		registerPubSubHandlers()
		cleanupOnce.Do(func() {
			go cleanupOrphanSubscriptionsLoop()
		})
	}
}

func registerPubSubHandlers() {
	ps := pubsub.GetPubSub()

	ps.RegisterHandler(pubsub.ChannelEvents, func(msg *pubsub.Message) error {
		var eventMsg pubsub.EventMessage
		if err := json.Unmarshal([]byte(msg.Payload), &eventMsg); err != nil {
			return err
		}
		if eventMsg.Event != nil {
			notifyLocalListeners(eventMsg.Event)
		}
		return nil
	})

	ps.RegisterHandler(pubsub.ChannelSubCreate, func(msg *pubsub.Message) error {
		var subMsg pubsub.SubCreateMessage
		if err := json.Unmarshal([]byte(msg.Payload), &subMsg); err != nil {
			return err
		}
		return handleRemoteSubCreate(subMsg.WSID, subMsg.SubID, subMsg.Filter)
	})

	ps.RegisterHandler(pubsub.ChannelSubClose, func(msg *pubsub.Message) error {
		var closeMsg pubsub.SubCloseMessage
		if err := json.Unmarshal([]byte(msg.Payload), &closeMsg); err != nil {
			return err
		}
		return handleRemoteSubClose(closeMsg.WSID, closeMsg.SubID)
	})

	ps.RegisterHandler(pubsub.ChannelWSDisconnect, func(msg *pubsub.Message) error {
		var disconnectMsg pubsub.WSConnectMessage
		if err := json.Unmarshal([]byte(msg.Payload), &disconnectMsg); err != nil {
			return err
		}
		handleRemoteDisconnect(disconnectMsg.WSID)
		return nil
	})

	ps.RegisterHandler(pubsub.ChannelSubCleanup, func(msg *pubsub.Message) error {
		var cleanupMsg pubsub.SubCleanupMessage
		if err := json.Unmarshal([]byte(msg.Payload), &cleanupMsg); err != nil {
			return err
		}
		handleRemoteDisconnect(cleanupMsg.WSID)
		return nil
	})
}

func notifyLocalListeners(event *nostr.Event) {
	localListenersMutex.RLock()
	defer localListenersMutex.RUnlock()

	for ws, subs := range localListeners {
		for id, listener := range subs {
			if listener.filters.Match(event) {
				select {
				case ws.ChanSender <- nostr.EventEnvelope{
					SubscriptionID: &id,
					Event:          *event,
				}:
					metrics.NostrEventsNotifiedCounter.Inc()
				default:
					log.Logger.Warn("failed to send event to listener buffer full",
						zap.String("ws_id", getWsID(ws)),
						zap.String("sub_id", id),
					)
				}
			}
		}
	}
}

func handleRemoteSubCreate(wsID, subID string, filter json.RawMessage) error {
	var filters nostr.Filters
	if err := json.Unmarshal(filter, &filters); err != nil {
		return err
	}

	key := fmt.Sprintf("subs:%s", wsID)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	data := ListenerData{
		Filters:   filter,
		CreatedAt: time.Now().Unix(),
	}
	dataJSON, _ := json.Marshal(data)

	redisClient := cache.GetRedis()
	if redisClient != nil {
		redisClient.HSet(ctx, key, subID, string(dataJSON))
	}

	return nil
}

func handleRemoteSubClose(wsID, subID string) error {
	key := fmt.Sprintf("subs:%s", wsID)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	redisClient := cache.GetRedis()
	if redisClient != nil {
		redisClient.HDel(ctx, key, subID)
	}

	return nil
}

func handleRemoteDisconnect(wsID string) {
	key := fmt.Sprintf("subs:%s", wsID)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	redisClient := cache.GetRedis()
	if redisClient != nil {
		redisClient.Del(ctx, key)
	}

	localListenersMutex.Lock()
	defer localListenersMutex.Unlock()

	for ws, id := range wsToID {
		if id == wsID {
			if subs, ok := localListeners[ws]; ok {
				listenerCount.Add(-int32(len(subs)))
				metrics.NostrListenerGauge.Sub(float64(len(subs)))
				delete(localListeners, ws)
			}
			wsToIDMutex.Lock()
			delete(wsToID, ws)
			wsToIDMutex.Unlock()
			break
		}
	}
}

func GetListeningFilters() nostr.Filters {
	localListenersMutex.RLock()
	defer localListenersMutex.RUnlock()

	respfilters := nostr.Filters{}
	uniqueFilters := make(map[string]struct{})

	for _, connListeners := range localListeners {
		for _, listener := range connListeners {
			for _, listenerFilter := range listener.filters {
				filterKey := listenerFilter.String()
				if _, exists := uniqueFilters[filterKey]; !exists {
					uniqueFilters[filterKey] = struct{}{}
					respfilters = append(respfilters, listenerFilter)
				}
			}
		}
	}
	return respfilters
}

func SetListener(id string, ws *dto.WsServer, filters nostr.Filters) {
	localListenersMutex.Lock()
	defer localListenersMutex.Unlock()

	if localListeners[ws] == nil {
		localListeners[ws] = make(map[string]*Listener)
		wsID := fmt.Sprintf("ws_%s_%d", ws.Conn.IP(), time.Now().UnixNano())
		setWsID(ws, wsID)
		touchWSID(wsID)

		if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
			go func() {
				ps.PublishWSConnect(context.Background(), wsID, ws.Authed)
			}()
		}

		if redisClient := cache.GetRedis(); redisClient != nil && cache.IsEnabled() {
			key := fmt.Sprintf("subs:%s", wsID)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			redisClient.HSet(ctx, key, "info", fmt.Sprintf(`{"ip":"%s","authed":"%s"}`, ws.Conn.IP(), ws.Authed))
		}
	}

	if _, exists := localListeners[ws][id]; !exists {
		listenerCount.Add(1)
		metrics.NostrConnectionCounter.Inc()
		metrics.NostrListenerGauge.Inc()
		metrics.NostrListenerAddCounter.Inc()
	}

	localListeners[ws][id] = &Listener{
		filters:   filters,
		createdAt: time.Now(),
	}

	wsID := getWsID(ws)
	if wsID != "" {
		touchWSID(wsID)
		filterJSON, _ := json.Marshal(filters)
		if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
			go func() {
				ps.PublishSubCreate(context.Background(), wsID, id, filterJSON)
			}()
		}

		if redisClient := cache.GetRedis(); redisClient != nil && cache.IsEnabled() {
			key := fmt.Sprintf("subs:%s", wsID)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			data := ListenerData{
				Filters:   filterJSON,
				CreatedAt: time.Now().Unix(),
			}
			dataJSON, _ := json.Marshal(data)
			redisClient.HSet(ctx, key, id, string(dataJSON))
		}
	}
}

func SubscriptionCount(ws *dto.WsServer) int {
	localListenersMutex.RLock()
	defer localListenersMutex.RUnlock()

	return subscriptionCountLocked(ws)
}

func HasSubscription(ws *dto.WsServer, id string) bool {
	localListenersMutex.RLock()
	defer localListenersMutex.RUnlock()

	if subs, ok := localListeners[ws]; ok {
		_, exists := subs[id]
		return exists
	}
	return false
}

func subscriptionCountLocked(ws *dto.WsServer) int {
	if subs, ok := localListeners[ws]; ok {
		return len(subs)
	}
	return 0
}

func RemoveListenerId(ws *dto.WsServer, id string) {
	localListenersMutex.Lock()
	defer localListenersMutex.Unlock()

	if subs, ok := localListeners[ws]; ok {
		if _, exists := subs[id]; exists {
			delete(subs, id)
			listenerCount.Add(-1)
			metrics.NostrConnectionCounter.Desc()
			metrics.NostrListenerGauge.Dec()
			metrics.NostrListenerRemoveCounter.Inc()

			wsID := getWsID(ws)
			if wsID != "" {
				if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
					go func() {
						ps.PublishSubClose(context.Background(), wsID, id)
					}()
				}

				if redisClient := cache.GetRedis(); redisClient != nil && cache.IsEnabled() {
					key := fmt.Sprintf("subs:%s", wsID)
					ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
					defer cancel()
					redisClient.HDel(ctx, key, id)
				}
			}
		}
		if len(subs) == 0 {
			delete(localListeners, ws)
		}
	}
}

func RemoveListener(ws *dto.WsServer) {
	localListenersMutex.Lock()
	defer localListenersMutex.Unlock()

	wsID := getWsID(ws)

	if subs, ok := localListeners[ws]; ok {
		removedCount := len(subs)
		delete(localListeners, ws)
		metrics.NostrListenerGauge.Sub(float64(removedCount))
		metrics.NostrListenerRemoveCounter.Add(float64(removedCount))
		metrics.NostrConnectionCounter.Sub(float64(removedCount))
		listenerCount.Add(-int32(removedCount))
	}

	if wsID != "" {
		if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
			go func() {
				ps.PublishWSDisconnect(context.Background(), wsID)
			}()
		}

		if redisClient := cache.GetRedis(); redisClient != nil && cache.IsEnabled() {
			key := fmt.Sprintf("subs:%s", wsID)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			redisClient.Del(ctx, key)
		}

		wsToIDMutex.Lock()
		delete(wsToID, ws)
		wsToIDMutex.Unlock()
	}
}

func Touch(ws *dto.WsServer) {
	wsID := getWsID(ws)
	if wsID == "" {
		return
	}
	ws.LastSeen = time.Now().UTC()
	touchWSID(wsID)
}

func touchWSID(wsID string) {
	if wsID == "" || !cache.IsEnabled() {
		return
	}
	redisClient := cache.GetRedis()
	if redisClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ttl := time.Duration(configuredStaleAfterSeconds()) * time.Second
	_ = redisClient.Set(ctx, wsLastSeenPrefix+wsID, fmt.Sprintf("%d", time.Now().Unix()), ttl)
	_ = redisClient.Expire(ctx, fmt.Sprintf("subs:%s", wsID), ttl)
}

func cleanupOrphanSubscriptionsLoop() {
	interval := time.Duration(configuredCleanupIntervalSeconds()) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		cleanupOrphanSubscriptions()
	}
}

func cleanupOrphanSubscriptions() {
	if !cache.IsEnabled() {
		return
	}
	redisClient := cache.GetRedis()
	if redisClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var cursor uint64
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "subs:*", 100)
		if err != nil {
			log.Logger.Debug("failed scanning subscriptions for cleanup", zap.Error(err))
			return
		}
		for _, key := range keys {
			wsID := strings.TrimPrefix(key, "subs:")
			exists, err := redisClient.Exists(ctx, wsLastSeenPrefix+wsID)
			if err != nil {
				continue
			}
			if exists == 0 {
				_ = redisClient.Del(ctx, key)
				metrics.NostrListenerOrphanCleanup.Inc()
				if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
					_ = ps.PublishSubCleanup(context.Background(), wsID)
				}
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return
		}
	}
}

func configuredCleanupIntervalSeconds() int {
	if config.Cfg == nil || config.Cfg.Redis.SubscriptionCleanupIntervalSeconds <= 0 {
		return 60
	}
	return config.Cfg.Redis.SubscriptionCleanupIntervalSeconds
}

func configuredStaleAfterSeconds() int {
	if config.Cfg == nil || config.Cfg.Redis.SubscriptionStaleAfterSeconds <= 0 {
		return 120
	}
	return config.Cfg.Redis.SubscriptionStaleAfterSeconds
}

func NotifyListeners(event *nostr.Event) {
	if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
		go func() {
			if err := ps.PublishEvent(context.Background(), event); err != nil {
				log.Logger.Debug("failed to publish event to redis", zap.Error(err))
			}
		}()
	}

	notifyLocalListeners(event)
}

func GetCount() int {
	return int(listenerCount.Load())
}

func ActiveConnections() []ConnectionInfo {
	localListenersMutex.RLock()
	defer localListenersMutex.RUnlock()

	connections := make([]ConnectionInfo, 0, len(localListeners))
	for ws, subs := range localListeners {
		connections = append(connections, ConnectionInfo{
			WSID:              getWsID(ws),
			IP:                ws.Conn.IP(),
			Authed:            ws.Authed,
			SubscriptionCount: len(subs),
			ConnectedAt:       ws.StartTime.Unix(),
			LastSeenAt:        ws.LastSeen.Unix(),
			UserAgent:         ws.UserAgent,
		})
	}
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].LastSeenAt == connections[j].LastSeenAt {
			return connections[i].WSID < connections[j].WSID
		}
		return connections[i].LastSeenAt > connections[j].LastSeenAt
	})
	return connections
}

func AuthedConnections() []ConnectionInfo {
	active := ActiveConnections()
	authed := make([]ConnectionInfo, 0, len(active))
	for _, conn := range active {
		if conn.Authed != "" {
			authed = append(authed, conn)
		}
	}
	return authed
}

func Disconnect(wsID string) bool {
	wsToIDMutex.RLock()
	var target *dto.WsServer
	for ws, id := range wsToID {
		if id == wsID {
			target = ws
			break
		}
	}
	wsToIDMutex.RUnlock()

	if target == nil || target.Conn == nil {
		return false
	}

	RemoveListener(target)
	_ = target.Conn.Close()
	return true
}

func MatchEventAgainstSubscribers(event *nostr.Event, wsID string) bool {
	localListenersMutex.RLock()
	defer localListenersMutex.RUnlock()

	for ws, subs := range localListeners {
		currentWsID := getWsID(ws)
		if currentWsID == wsID {
			for id, listener := range subs {
				if listener.filters.Match(event) {
					return true
				}
				_ = id
			}
		}
	}
	return false
}
