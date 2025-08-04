# NostrPool

`NostrPool` é uma biblioteca Go que gerencia um pool de conexões com relays [Nostr](https://github.com/nostr-protocol/nostr). Ela simplifica a interação com múltiplos relays, agregando eventos de inscrições e distribuindo publicações de forma concorrente e resiliente.

Este pacote utiliza um padrão Singleton para o pool, garantindo que uma única instância gerencie todas as conexões de relays durante o ciclo de vida da aplicação.

## ✨ Funcionalidades

- **Gerenciamento de Múltiplos Relays**: Conecte-se a vários relays Nostr simultaneamente.
- **Padrão Singleton**: Uma única instância do pool é compartilhada em toda a aplicação.
- **Publicação e Inscrição Simplificadas**: Funções `Publish` e `Subscribe` de alto nível que operam em todos os relays conectados.
- **Agregação de Eventos**: Receba eventos de todos os relays em um único canal Go (`chan`).
- **Reconexão Automática**: Tenta reconectar-se a um relay se uma operação de publicação ou inscrição falhar.
- **Manipulação de Timeouts**: As operações de rede possuem timeouts para evitar bloqueios indefinidos.

## 📦 Instalação

```sh
go get github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool
```
*(**Nota**: Substitua `SEU-USUARIO/nostrpool` pelo caminho de importação real do seu pacote).*

## 🚀 Uso

A utilização do pool é simples e direta. Primeiro, inicialize o pool com os relays desejados e, em seguida, use as funções `Publish` e `Subscribe`.

### Exemplo Completo

Abaixo está um exemplo completo que demonstra como inicializar o pool, publicar um evento e se inscrever para receber novos eventos.

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/SEU-USUARIO/nostrpool" // Substitua pelo caminho correto do seu pacote
)

func main() {
	// 1. Inicialize o pool com os URLs dos relays desejados
	// A função Init pode retornar um erro se a conexão com o primeiro relay falhar,
	// mas continuará tentando conectar-se aos outros.
	relayURLs := []string{
		"wss://relay.damus.io",
		"wss://relay.snort.social",
		"wss://nos.lol",
	}
	if err := nostrpool.Init(relayURLs); err != nil {
		log.Printf("Erro ao inicializar com alguns relays, mas continuando: %v", err)
	}

	// --- Publicando um Evento ---

	// Para publicar, você precisa de uma chave privada.
	// Em uma aplicação real, carregue isso de uma configuração segura.
	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)
	npub, _ := nip19.EncodePublicKey(pk)

	fmt.Printf("Publicando como: %s\n", npub)

	ev := nostr.Event{
		PubKey:    pk,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Content:   "Olá, Nostr, a partir do NostrPool! " + time.Now().Format(time.RFC3339),
		Tags:      nostr.Tags{{"t", "nostrpool-example"}},
	}

	// Assina o evento com a chave privada
	if err := ev.Sign(sk); err != nil {
		log.Fatalf("Erro ao assinar o evento: %v", err)
	}

	// Publica o evento em todos os relays do pool
	if err := nostrpool.Publish(&ev); err != nil {
		// Retorna o primeiro erro encontrado, mas tenta publicar em todos os relays
		log.Printf("Erro ao publicar em pelo menos um relay: %v", err)
	} else {
		fmt.Println("Evento publicado com sucesso em todos os relays conectados!")
	}

	fmt.Println("\n---\n")
	time.Sleep(2 * time.Second) // Pausa para garantir que o evento foi propagado

	// --- Inscrevendo-se para Receber Eventos ---

	fmt.Println("Inscrevendo-se para receber eventos de notas de texto...")
	filters := nostr.Filters{
		{
			Kinds: []int{nostr.KindTextNote},
			Tags:  nostr.TagMap{"#t": []string{"nostrpool-example"}},
			Limit: 5, // Limita a 5 eventos por relay
		},
	}

	// O canal de eventos unificado
	eventChan, err := nostrpool.Subscribe(filters)
	if err != nil {
		log.Fatalf("Erro ao se inscrever: %v", err)
	}

	fmt.Println("Aguardando eventos... (o programa será encerrado em 15 segundos)")

	// Processa os eventos recebidos do canal unificado
	// O canal será fechado quando o contexto da inscrição expirar (10s) ou
	// quando todas as goroutines de subscrição terminarem.
	for event := range eventChan {
		fmt.Printf("Evento recebido (ID: %.10s...): %s\n", event.ID, event.Content)
	}

	fmt.Println("Inscrição encerrada.")
}
```

## 📖 API

### `Init(relayURLs []string) error`
Inicializa o pool de relays singleton. Esta função deve ser chamada **apenas uma vez** no início da sua aplicação.

- `relayURLs`: Uma lista de strings contendo os URLs dos relays (`wss://...`).
- A função tenta se conectar a todos os relays fornecidos.
- Retorna o erro da primeira conexão que falhar, mas continua tentando conectar-se aos outros. Se nenhum relay puder ser conectado, um erro é retornado.

### `Publish(ev *nostr.Event) error`
Publica um evento `nostr.Event` em todos os relays conectados no pool.

- `ev`: Um ponteiro para o evento a ser publicado. O evento já deve estar assinado.
- A função publica o evento de forma concorrente em todos os relays.
- Se a publicação falhar em um relay, ele tentará reconectar e publicar novamente.
- Retorna o primeiro erro encontrado durante o processo. Um valor de retorno `nil` não garante que a publicação foi bem-sucedida em todos os relays, mas indica que não houve erros imediatos.

### `Subscribe(filters nostr.Filters) (<-chan *nostr.Event, error)`
Inscreve-se para receber eventos com base nos filtros fornecidos em todos os relays do pool.

- `filters`: Um `nostr.Filters` que define os critérios para os eventos desejados.
- Retorna um canal unificado (`<-chan *nostr.Event`) que receberá eventos de todos os relays.
- O canal é fechado quando o contexto da inscrição (com timeout de 10 segundos) é cancelado ou quando todas as subscrições individuais são encerradas.
- Se ocorrer um erro ao se inscrever em um relay, ele tentará reconectar e se inscrever novamente.

## ⚙️ Concorrência e Tratamento de Erros

- **Concorrência**: O pacote é projetado para ser "thread-safe". As operações de publicação e inscrição são executadas em goroutines separadas para cada relay, e o acesso ao pool é protegido por um `sync.Mutex`.
- **Tratamento de Erros**:
    - `Init` e `Publish` retornam o **primeiro erro** que encontrarem. Isso significa que a operação pode ter sido bem-sucedida em alguns relays, mesmo que um erro seja retornado.
    - `Subscribe` retornará um erro se não houver relays conectados ou se a inicialização falhar.

## 🤝 Dependências

- [github.com/nbd-wtf/go-nostr](https://github.com/nbd-wtf/go-nostr): A biblioteca base para o protocolo Nostr em Go.

## 📜 Licença

Este pacote é distribuído sob a licença MIT.