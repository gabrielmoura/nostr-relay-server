package nostrpool

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type RelayPool struct {
	relays   map[string]*nostr.Relay
	mu       sync.Mutex
	initOnce sync.Once
}

var (
	pool     *RelayPool
	poolOnce sync.Once
)

// Init inicializa o pool singleton com múltiplos relays.
// Deve ser chamado uma vez no início da aplicação.
func Init(relayURLs []string) error {
	var err error
	poolOnce.Do(func() {
		pool = &RelayPool{
			relays: make(map[string]*nostr.Relay),
		}
		err = pool.connectAll(relayURLs)
	})
	return err
}

// connectAll conecta a todos os relays fornecidos, armazenando as conexões.
func (p *RelayPool) connectAll(urls []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		r, err := nostr.RelayConnect(ctx, url)
		cancel()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		p.relays[url] = r
	}
	if len(p.relays) == 0 {
		return errors.New("falha ao conectar a todos os relays")
	}
	return firstErr
}

// reconnectRelay tenta reconectar um relay específico.
func (p *RelayPool) reconnectRelay(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		return err
	}
	p.relays[url] = r
	return nil
}

// Subscribe cria inscrições em todos os relays com o filtro fornecido.
// Retorna um canal unificado de eventos e erro.
func Subscribe(filters nostr.Filters) (<-chan *nostr.Event, error) {
	if pool == nil {
		return nil, errors.New("pool não inicializado, chame Init() primeiro")
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
		return nil, errors.New("nenhum relay conectado")
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
		return errors.New("pool não inicializado, chame Init() primeiro")
	}

	pool.mu.Lock()
	relays := make([]*nostr.Relay, 0, len(pool.relays))
	for _, r := range pool.relays {
		relays = append(relays, r)
	}
	pool.mu.Unlock()

	if len(relays) == 0 {
		return errors.New("nenhum relay conectado")
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
