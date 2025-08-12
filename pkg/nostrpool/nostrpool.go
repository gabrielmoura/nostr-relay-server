package nostrpool

import (
	"context"
	"errors"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type RelayPool struct {
	relays        map[string]*nostr.Relay
	mu            sync.Mutex
	initOnce      sync.Once
	relayFailures map[string]int // URL -> failure count
	context.Context
}

const MaxFailures = 5 // Número máximo de falhas antes de penalizar um relay

var (
	pool                  *RelayPool
	poolOnce              sync.Once
	ErrMaxConnectFailure  = errors.New("máximo de erros ao conectar")
	ErrPollNotInitialized = errors.New("pool não inicializado, chame Init() primeiro")

	ErrNotRelayConnected    = errors.New("nenhum relay conectado")
	ErrNotRelayConnectedAll = errors.New("falha ao conectar a todos os relays")
)

// Init inicializa o pool singleton com múltiplos relays.
// Deve ser chamado uma vez no início da aplicação.
func Init(ctx context.Context, relayURLs []string) error {
	var err error
	poolOnce.Do(func() {
		pool = &RelayPool{
			relays:        make(map[string]*nostr.Relay),
			relayFailures: make(map[string]int),
			Context:       ctx,
		}
		err = pool.connectAll(relayURLs)
	})
	return err
}

// connectAll conecta a todos os relays fornecidos, armazenando as conexões.
func (p *RelayPool) connectAll(urls []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	headers := http.Header{}
	headers.Set("User-Agent", config.Cfg.RelayInformation.Name+"/"+config.Cfg.RelayInformation.Version)
	headers.Set("X-Nostr-Relay-Contact", config.Cfg.RelayInformation.Contact)

	var firstErr error
	for _, url := range urls {
		r, err := nostr.RelayConnect(p.Context, url, nostr.WithRequestHeader(headers))

		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			// Se falhar, adiciona ao relayFailures
			p.relayFailures[url]++
			continue
		}
		p.relays[url] = r
	}
	if len(p.relays) == 0 {
		return ErrNotRelayConnectedAll
	}
	return firstErr
}

// reconnectRelay tenta reconectar um relay específico.
func (p *RelayPool) reconnectRelay(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.relayFailures[url] >= MaxFailures {
		log.Logger.Warn("Removing Relay from Pool", zap.String("url", url), zap.Error(ErrMaxConnectFailure))
		delete(p.relays, url)
		return ErrMaxConnectFailure
	}
	r, err := nostr.RelayConnect(p.Context, url)
	if err != nil {
		p.relayFailures[url]++
		return err
	}
	p.relays[url] = r
	return nil
}

// Subscribe cria inscrições em todos os relays com o filtro fornecido.
// Retorna um canal unificado de eventos e erro.
func Subscribe(filters nostr.Filters) (<-chan *nostr.Event, error) {
	if pool == nil {
		return nil, ErrPollNotInitialized
	}

	out := make(chan *nostr.Event)
	var wg sync.WaitGroup

	pool.mu.Lock()
	relays := make([]*nostr.Relay, 0, len(pool.relays))
	for _, r := range pool.relays {
		relays = append(relays, r)
	}
	pool.mu.Unlock()

	if len(relays) == 0 {
		return nil, ErrNotRelayConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, r := range relays {
		wg.Add(1)
		go func(relay *nostr.Relay) {
			defer wg.Done()
			sub, err := relay.Subscribe(ctx, filters)
			if err != nil {
				// tenta reconectar e tentar novamente
				_ = pool.reconnectRelay(relay.URL)
				sub, err = relay.Subscribe(ctx, filters)
				if err != nil {
					return
				}
			}
			for ev := range sub.Events {
				out <- ev
			}
		}(r)
	}

	// Fecha o canal quando todas as goroutines terminarem
	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}

// Publish envia o evento para todos os relays com timeout e tratamento de erro.
func Publish(ev *nostr.Event) error {
	if pool == nil {
		return ErrPollNotInitialized
	}

	pool.mu.Lock()
	relays := make([]*nostr.Relay, 0, len(pool.relays))
	for _, r := range pool.relays {
		relays = append(relays, r)
	}
	pool.mu.Unlock()

	if len(relays) == 0 {
		return ErrNotRelayConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for _, r := range relays {
		wg.Add(1)
		go func(relay *nostr.Relay) {
			defer wg.Done()
			err := relay.Publish(ctx, *ev)
			if err != nil {
				// tenta reconectar e tentar novamente
				if recErr := pool.reconnectRelay(relay.URL); recErr == nil {
					err = relay.Publish(ctx, *ev)
				}
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}
		}(r)
	}

	wg.Wait()
	return firstErr
}
