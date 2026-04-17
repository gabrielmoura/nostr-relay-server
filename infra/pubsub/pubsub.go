package pubsub

import (
	"context"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	ChannelEvents       = "events"
	ChannelWSConnect    = "ws:connect"
	ChannelWSDisconnect = "ws:disconnect"
	ChannelSubCreate    = "sub:create"
	ChannelSubClose     = "sub:close"
	ChannelSubCleanup   = "sub:cleanup"
)

type EventMessage struct {
	Type  string       `json:"type"`
	Event *nostr.Event `json:"event,omitempty"`
}

type WSConnectMessage struct {
	WSID   string `json:"ws_id"`
	PubKey string `json:"pubkey,omitempty"`
}

type SubCreateMessage struct {
	WSID   string          `json:"ws_id"`
	SubID  string          `json:"sub_id"`
	Filter json.RawMessage `json:"filter"`
}

type SubCloseMessage struct {
	WSID  string `json:"ws_id"`
	SubID string `json:"sub_id"`
}

type SubCleanupMessage struct {
	WSID string `json:"ws_id"`
}

type PubSub struct {
	client   *goredis.Client
	handlers map[string]MessageHandler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	enabled  bool
}

type Subscriber struct {
	ch    <-chan *goredis.Message
	unsub func()
	wsID  string
}

type MessageHandler func(msg *Message) error

type Message struct {
	Channel string
	Payload string
}

var (
	pubsubInstance *PubSub
	pubsubOnce     sync.Once
)

func Init() error {
	var initErr error
	pubsubOnce.Do(func() {
		client := redis.GetClient()
		if client == nil || !client.IsEnabled() {
			log.Logger.Info("Redis pub/sub is disabled")
			pubsubInstance = &PubSub{enabled: false}
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		pubsubInstance = &PubSub{
			client:   client.Raw(),
			handlers: make(map[string]MessageHandler),
			ctx:      ctx,
			cancel:   cancel,
			enabled:  true,
		}

		if err := pubsubInstance.startListening(); err != nil {
			cancel()
			initErr = err
			return
		}

		log.Logger.Info("Redis pub/sub initialized")
	})
	return initErr
}

func GetPubSub() *PubSub {
	return pubsubInstance
}

func (p *PubSub) startListening() error {
	channels := []string{
		ChannelEvents,
		ChannelWSConnect,
		ChannelWSDisconnect,
		ChannelSubCreate,
		ChannelSubClose,
		ChannelSubCleanup,
	}

	for _, ch := range channels {
		pubsub := p.client.Subscribe(p.ctx, ch)
		if pubsub == nil {
			continue
		}

		chName := ch
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case msg, ok := <-pubsub.Channel():
					if !ok {
						return
					}
					p.handleMessage(chName, msg)
				case <-p.ctx.Done():
					pubsub.Close()
					return
				}
			}
		}()
	}

	return nil
}

func (p *PubSub) handleMessage(channel string, msg *goredis.Message) {
	p.mu.RLock()
	handler, ok := p.handlers[channel]
	p.mu.RUnlock()

	if ok && handler != nil {
		customMsg := &Message{
			Channel: msg.Channel,
			Payload: msg.Payload,
		}
		if err := handler(customMsg); err != nil {
			log.Logger.Error("pubsub handler error",
				zap.String("channel", channel),
				zap.Error(err),
			)
		}
	}
}

func (p *PubSub) RegisterHandler(channel string, handler MessageHandler) {
	p.mu.Lock()
	p.handlers[channel] = handler
	p.mu.Unlock()
}

func (p *PubSub) PublishEvent(ctx context.Context, event *nostr.Event) error {
	if !p.enabled || p.client == nil {
		return nil
	}

	msg := EventMessage{
		Type:  "event",
		Event: event,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, ChannelEvents, string(data)).Err()
}

func (p *PubSub) PublishWSConnect(ctx context.Context, wsID, pubKey string) error {
	if !p.enabled || p.client == nil {
		return nil
	}

	msg := WSConnectMessage{
		WSID:   wsID,
		PubKey: pubKey,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, ChannelWSConnect, string(data)).Err()
}

func (p *PubSub) PublishWSDisconnect(ctx context.Context, wsID string) error {
	if !p.enabled || p.client == nil {
		return nil
	}

	msg := WSConnectMessage{
		WSID: wsID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, ChannelWSDisconnect, string(data)).Err()
}

func (p *PubSub) PublishSubCreate(ctx context.Context, wsID, subID string, filter json.RawMessage) error {
	if !p.enabled || p.client == nil {
		return nil
	}

	msg := SubCreateMessage{
		WSID:   wsID,
		SubID:  subID,
		Filter: filter,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, ChannelSubCreate, string(data)).Err()
}

func (p *PubSub) PublishSubClose(ctx context.Context, wsID, subID string) error {
	if !p.enabled || p.client == nil {
		return nil
	}

	msg := SubCloseMessage{
		WSID:  wsID,
		SubID: subID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, ChannelSubClose, string(data)).Err()
}

func (p *PubSub) PublishSubCleanup(ctx context.Context, wsID string) error {
	if !p.enabled || p.client == nil {
		return nil
	}

	msg := SubCleanupMessage{WSID: wsID}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, ChannelSubCleanup, string(data)).Err()
}

func (p *PubSub) Close() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *PubSub) IsEnabled() bool {
	return p.enabled && p.client != nil
}
