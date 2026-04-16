# Negentropy Go Library

## Visao geral

`negentropy_go` e uma biblioteca Go para implementar o lado de processamento do NIP-77 sem dependencia de transporte.

Ela foi desenhada para o mesmo modelo usado no projeto atual: a camada de rede recebe mensagens, converte para tipos internos e delega a reconciliacao para um componente de dominio.

## Objetivo da biblioteca

- Implementar o comportamento esperado pelo NIP-77 e pelo protocolo Negentropy V1.
- Permitir integracao com qualquer storage via interfaces pequenas.
- Reutilizar consultas para evitar trabalho duplicado em cargas concorrentes.

## Responsabilidades e limites

Responsabilidades da biblioteca:

- Processar payloads `NEG-OPEN` e `NEG-MSG`.
- Manter estado de sessao de reconciliacao.
- Executar reconciliacao usando conjunto de eventos retornado pelo storage.
- Expor respostas de dominio (`NEG-MSG`/`NEG-ERR`) para a camada externa enviar.

Limites (fora da biblioteca):

- Abertura de conexao WebSocket.
- Parse de JSON da mensagem Nostr de transporte.
- Controle de autenticacao/autorizacao.
- Implementacao concreta de banco de dados.

## Estrutura de pastas e pacotes

- `negentropy_go/model`: tipos, filtros, requests/responses.
- `negentropy_go/contracts`: interfaces de storage e cache.
- `negentropy_go/cache`: cache em memoria + chave normalizada de filtro.
- `negentropy_go/protocol`: varint, bound, range e codec binario.
- `negentropy_go/engine`: reconciliador puro (algoritmo Negentropy).
- `negentropy_go/service`: gerenciamento de sessao, cache e fluxo de negocio.
- `negentropy_go/examples/inmemory`: exemplo executavel sem transporte.

## Interfaces principais

```go
type EventStore interface {
    QueryEventRefs(ctx context.Context, filter model.Filter) ([]model.EventRef, error)
}

type QueryCache interface {
    Get(key string) ([]model.EventRef, bool)
    Set(key string, refs []model.EventRef, ttl time.Duration)
    Delete(key string)
    PurgeExpired(now time.Time)
}
```

`EventRef` contem apenas os dados necessarios para Negentropy (`created_at` e `id`), reduzindo acoplamento com o modelo completo de evento.

## Fluxo de funcionamento

1. Camada externa recebe JSON Nostr (`NEG-OPEN`, `NEG-MSG`, `NEG-CLOSE`).
2. Traduz para `model.OpenRequest` ou `model.MessageRequest`.
3. Chama `Manager.Open` ou `Manager.OnMessage`.
4. `Manager` busca refs no `EventStore`, usando cache + deduplicacao de query concorrente.
5. `engine.Reconciler` processa o protocolo e devolve payload binario.
6. Camada externa converte para hex e envia `NEG-MSG` para o cliente.
7. Em erro de protocolo/sessao, retorna `NEG-ERR` com motivo.

## Reaproveitamento de buscas e concorrencia

- A chave de cache usa filtro normalizado (`cache.BuildFilterKey`) para evitar misses por ordenacao diferente.
- `Manager` deduplica carregamentos simultaneos da mesma chave com controle de inflight (`loadOnce`).
- O cache e thread-safe.
- Sessao e cache tem expurgo por TTL (`PurgeExpired`).

## Exemplo de integracao com WebSocket externo (Fiber)

Exemplo de adaptacao da camada externa usando `github.com/gofiber/contrib/websocket`:

```go
package main

import (
    "context"
    "encoding/json"

    "github.com/gofiber/contrib/websocket"
    negentropy "github.com/hoytech/strfry/negentropy_go"
)

func wsHandler(manager *negentropy.Manager) func(*websocket.Conn) {
    return func(c *websocket.Conn) {
        defer c.Close()

        for {
            _, payload, err := c.ReadMessage()
            if err != nil {
                return
            }

            var msg []any
            if err := json.Unmarshal(payload, &msg); err != nil {
                continue
            }

            verb, _ := msg[0].(string)

            switch verb {
            case "NEG-OPEN":
                req := translateOpen(msg)
                resp, err := manager.Open(context.Background(), req)
                if err != nil {
                    continue
                }
                writeNegResponse(c, resp)

            case "NEG-MSG":
                req := translateMsg(msg)
                resp, err := manager.OnMessage(context.Background(), req)
                if err != nil {
                    continue
                }
                writeNegResponse(c, resp)

            case "NEG-CLOSE":
                sessionID, _ := msg[1].(string)
                manager.Close(sessionID)
            }
        }
    }
}
```

### Como a camada externa deve operar

- Receber mensagem via WebSocket.
- Validar e traduzir JSON para tipos da biblioteca (`OpenRequest`, `MessageRequest`).
- Acionar `Manager.Open`/`Manager.OnMessage`.
- Persistir/consultar eventos via implementacao do `EventStore`.
- Enviar resposta Nostr com `NEG-MSG` ou `NEG-ERR` conforme `model.Response`.

## Exemplo de uso em projeto existente

O exemplo `negentropy_go/examples/inmemory/main.go` mostra o fluxo completo de:

- criacao do `Manager`
- injeção de `EventStore`
- montagem de mensagem inicial
- chamada de `Open`
- processamento da resposta

Esse mesmo padrao pode ser acoplado ao pipeline atual de ingestao/roteamento sem introduzir acoplamento com o framework de rede.

## Decisoes arquiteturais relevantes

- Separacao explicita de protocolo, algoritmo e orquestracao de sessao.
- Contratos pequenos para storage/cache.
- API sem dependencia de websocket/http/db.
- Concurrency com ganho real: deduplicacao de query concorrente e acesso thread-safe a estado.

## Pontos de extensao futura

- Cache distribuido (Redis/Memcached) via `QueryCache`.
- Estrategias alternativas de particionamento (`Buckets`) por perfil de carga.
- Suporte a limites de frame mais sofisticados e telemetria detalhada.
- Adicao de modo iniciador de alto nivel no `service` para cenarios relay-relay.

## Limitacoes conhecidas

- O `service.Manager` atual implementa o lado servidor/provider do fluxo NIP-77.
- O parser assume payload binario em hex valido vindo da camada externa.
- Diffs duplicados podem ocorrer quando `FrameSizeLimit` for usado agressivamente (comportamento esperado pelo protocolo).
