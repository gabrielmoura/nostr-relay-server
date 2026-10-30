# RelaySync v2: plano de implementação

Status: core, transportes genéricos e ciclo finito implementados no pacote;
integração PostgreSQL/Transactional Outbox do NRServer ainda pendente.

## Regras de execução

- Diretório da biblioteca: `pkg/relaySync`; cláusula: `package relaysync`.
- Permanecer no `go.mod` raiz; nenhum módulo aninhado ou `replace` local.
- Preservar o NIP-77 e o worktree preexistente.
- Executar uma fase por vez, atualizar `todo.md`, testar e revisar antes de
  avançar.
- Não iniciar integração NRServer antes dos gates do core e transports.
- Não habilitar nenhuma feature por default.
- Mudança destrutiva, quebra pública ou alteração de garantia exige nova
  decisão antes de editar.

## Baseline auditado

| Item | Estado encontrado |
| --- | --- |
| Go | `go 1.26.3` no módulo |
| HTTP | Fiber 2.52.12/fasthttp no listener existente |
| Nostr | go-nostr 0.52.3 |
| Storage | pgx/pgxpool 5.9.2, PostgreSQL |
| Redis | go-redis 9.18.0 para cache/PubSub/fila, não journal |
| Protobuf | runtime 1.36.11 indireto; não há `.proto` |
| zstd | klauspost/compress 1.18.5 indireto |
| Tooling | `protoc` 35.0, `gopls` e `staticcheck`; Buf ausente |
| CI | build, generated GraphQL check e `go test -race -shuffle=on ./...` |
| NIP-77 | `pkg/negentropy`, `pkg/negentropyV2`, `internal/sync` |
| RelaySync | inexistente |

Gaps conhecidos: Negentropy legado materializa slices inteiros, sessões são
locais, ACK de ingestão não equivale a commit, não há snapshot versionado,
journal, checkpoint RelaySync, golden Protobuf ou benchmarks de escala.

## Fase 0 — auditoria e arquitetura

Entregas:

- analisar instruções, módulo, CI, router, WS, NIP-77, policies, storage,
  observabilidade e testes;
- consultar NIP-01/09/11/40/42/77/98, Negentropy, Protobuf e zstd;
- criar os seis documentos em `docs/relaysync/`;
- registrar decisões, gates e riscos.

Critério de aceite:

- documentos coerentes entre si;
- nenhuma alteração funcional;
- path `pkg/relaySync` e coexistência registrados;
- aprovação explícita do usuário.

Resultado: concluída. A avaliação adicional de Vector/BTreeMem/BTreeLMDB/
SubRange e HORNET go-negentropy está em `negentropy-storage-evaluation.md`.

## Fase 1 — domínio, contracts e Protobuf

Escopo:

1. Criar `pkg/relaySync`, `doc.go`, README e tipos de domínio.
2. Implementar serialização canônica NIP-01 e validação de tamanhos.
3. Definir ports de snapshot, escrita, sessão, journal e checkpoint.
4. Criar stores em memória para testes, com expiração e monotonicidade.
5. Criar `relaysync.proto`, geração fixada e descriptor golden.
6. Implementar conversão Protobuf/domínio com limites explícitos.
7. Criar erros tipados e opções/defaults validados.

Dependências novas propostas:

- `google.golang.org/protobuf` passa a direta;
- `protoc-gen-go` é fixado como tool do módulo.

Não inclui: HTTP, Fiber, PostgreSQL, Redis, integração ou live real.

Testes/gates:

- roundtrip e golden Protobuf;
- evolução do descriptor;
- ID canônico NIP-01, assinatura/tamanhos e tags ordenadas;
- session/checkpoint monotônicos e expiração;
- fuzz de converters e framing básico;
- `gopls check`, `go test`, race dos packages novos e `go vet` focado;
- nenhum import de `internal/*`, `infra/*`, Fiber, pgx ou Redis pelo core.

Resultado: concluída para domínio, ports, stores de sessão/checkpoint,
conversores e schema Protobuf. Negentropy, framing, transportes e stores de
snapshot/journal permanecem deliberadamente nas fases seguintes.

## Fase 2 — manifest, snapshots e Negentropy

Escopo:

1. Implementar partição temporal/prefixo e digest canônico.
2. Implementar manifest paginado e cache por versão.
3. Criar porta privada `internal/reconcile`.
4. Aplicar a decisão documentada em `negentropy-storage-evaluation.md`:
   Vector selado por partição, SubRange lógico e agregados incrementais.
5. Implementar Negentropy V1 independentemente em
   `internal/negentropyv1`, a partir da especificação pública; usar packages
   existentes somente como referência comportamental, sem import ou cópia.
6. Implementar rounds, diff, cancelamento e limites por partição.
7. Executar harness oficial C++/Go e vetores golden.
8. Criar snapshot/store materializado em memória/disco temporário para testes.

Gate da implementação Negentropy:

- revisão comprova que nenhum código de package externo foi incorporado;
- wire format segue exclusivamente o contrato V1 `0x61`;
- saída byte a byte compatível com referência V1;
- parser fuzzado sem panic/alocação não limitada;
- benchmark não pior que a implementação interna atual no dataset definido.

Se falhar, substituir apenas a implementação privada; a API pública não muda.

Critério de aceite:

- conjuntos iguais, parciais e vazios reconciliam;
- somente partições divergentes abrem diálogo;
- snapshot permanece estável sob escrita concorrente;
- nenhum dataset completo do relay é carregado em RAM;
- memória máxima é função do limite de partição/batch.

## Fase 3 — sincronização HTTP e zstd

Escopo:

1. Implementar handler e cliente `net/http` com `http.Client` injetável.
2. Implementar os endpoints definidos em `protocol.md`.
3. Reutilizar conexões e propagar context/deadlines.
4. Implementar pull/push paginados e idempotência.
5. Tornar klauspost/zstd dependência direta e criar codec limitado.
6. Implementar backpressure, semáforos e limites de request/batch.
7. Testar duas instâncias em memória ponta a ponta.

HTTP/2 é recomendado no transporte configurado pelo consumidor. O SDK não cria
cliente por request, servidor, TLS ou certificado.

Critério de aceite:

- dois relays de teste convergem usando somente HTTP;
- retry não duplica efeito;
- corpo malformado/bomba zstd é rejeitado antes de exceder limites;
- cancelamento libera conexão, iterator e goroutine;
- benchmarks com compressão off, nível 1 e nível 3 documentados.

## Fase 4 — journal, checkpoints e WebSocket live

Escopo do core:

1. Implementar `changes:next` para catch-up HTTP finito/HTTP-only.
2. Implementar framing binário, hello, subscribe, delta, ACK, flow control,
   heartbeat e protocol errors.
3. Implementar replay a partir de cursor, gap/epoch e `ReconcileRequired`.
4. Implementar writer único e filas bounded.
5. Implementar reconexão e persistência de checkpoint.
6. Criar journal durável de teste com injeção de falhas.

Critério de aceite:

- snapshot watermark → replay → live sem gap;
- ACK somente após decisão durável;
- queda antes/depois de ACK é recuperada idempotentemente;
- cliente lento não causa crescimento ilimitado;
- mudança de epoch/retenção exige nova reconciliação;
- bidirecional simultâneo não entra em echo loop nem perde eventos.

Isso valida o contrato genérico, não habilita live no NRServer.

## Fase 5 — adapters genéricos e documentação pública

Escopo:

- consolidar adapter `net/http`;
- implementar adapter Fiber 2.52.12 sem duplicar lógica;
- implementar upgrade WebSocket binário no adapter apropriado;
- exemplos compiláveis net/http, Fiber e storage em memória;
- suíte pública de conformidade dos quatro ports;
- README completo: segurança, limites, consistência, erros e performance.

O adapter Fiber registra rotas e converte contexto/headers/body. TLS/HTTP/2
terminam no roteador/proxy; não haverá segundo servidor obrigatório.

Critério de aceite:

- mesmo engine passa E2E nos dois adapters;
- exemplos compilam;
- core continua sem imports de framework;
- documentação não menciona APIs inexistentes.

## Fase 6 — integração NRServer

Esta fase exige nova leitura do código então vigente.

### 6A — HTTP snapshot-only

- adapter explícito entre `relaysync.Event` e `nostr.Event`;
- fachada de leitura pgx com scope/autorização;
- snapshot materializado em transação `REPEATABLE READ` limitada ao build;
- preservar refs e payloads durante TTL, sem transação entre requests;
- rotas `/sync/v2/*` no Fiber externo antes do fallback `/`;
- configuração Viper/YAML, métricas e lifecycle/shutdown;
- gates `enabled`, `snapshot_enabled`, `apply_enabled`, `catchup_enabled` e
  `live_enabled` false por default;
- terminação SSL/HTTP2 documentada no roteador/proxy.

Gate: snapshot HTTP, sessão, idempotência e checkpoint funcionam em múltiplas
réplicas sem afinidade; NIP-77 fica inalterado.

### 6B — journal durável com Transactional Outbox

Decisão aprovada: Transactional Outbox no PostgreSQL, com cada evento/mudança e
outbox gravados no mesmo commit. Um projector publica no journal durável,
confirma a outbox somente após publicação segura e aplica retenção limitada.
Redis Pub/Sub/NOTIFY apenas acorda leitores.

Antes de migration:

- documentar schema, índices, backfill, epoch inicial, rollback e locks;
- medir WAL retido, lag e throughput;
- revisar operação do slot e alertas;
- provar cobertura de insert, batch, replace, soft/hard delete e NIP-40.

- remover entradas outbox apenas após publicação idempotente e recuperável;
- provar que o ciclo `complete` remove a sessão e todo o estado temporário.

Gate: restart, replay, commit fora de ordem, retenção e gap testados. Live ainda
fica false.

### 6C — escrita e live experimentais

- fachada síncrona que reutiliza assinatura, policy e persistência atuais;
- resultado individual pós-commit;
- checkpoint store persistente;
- Fiber/WebSocket no endpoint dedicado;
- allowlist de peers e scopes mínimos;
- readiness separada para HTTP/journal/live;
- extensão NIP-11 privada somente quando saudável.

Gate: canary/soak e testes de falha; habilitação de live requer decisão
explícita.

## Fase 7 — performance e estabilização

Datasets gerados deterministicamente:

- 100 mil e 1 milhão no CI/manual recorrente conforme custo;
- 10 milhões somente sob demanda.

Divergências: 0%, 0,1%, 1%, 10%, 50% e 100%.

Métricas coletadas separadamente:

- enumeração, manifest, Negentropy, transferência, zstd, validação e escrita;
- duração, events/s, bytes, requests, rounds;
- `allocs/op`, heap/peak RSS, CPU, GC e DB time;
- live throughput, lag, fila e recuperação após queda.

Fluxo de otimização: baseline → pprof/trace → hipótese → mudança isolada →
benchstat → race/regressão. Não usar `unsafe`/cgo sem nova justificativa.

Critério de aceite:

- resultados e ambiente registrados;
- limites/defaults derivados de evidência;
- testes completos/race/static analysis passam;
- riscos e capacidades experimentais separados das concluídas;
- revisão de produção e tutorial de uso concluídos.

## Estratégia de validação

Comandos serão alinhados aos scripts existentes, com cache gravável em `/tmp`:

```bash
GOCACHE=/tmp/nrserver-go-build go test ./pkg/relaySync/...
GOCACHE=/tmp/nrserver-go-build go test -race ./pkg/relaySync/...
GOCACHE=/tmp/nrserver-go-build go vet ./pkg/relaySync/...
GOCACHE=/tmp/nrserver-go-build go test -race -shuffle=on ./...
```

O último comando exige os pré-requisitos normais do repositório, incluindo
submódulo I2P e assets gerados do dashboard. Falha ambiental será diferenciada
de falha do código, nunca silenciada.

## Decisões que exigem aprovação posterior

1. Perfil/limites da implementação Negentropy independente após benchmarks.
2. Perfil operacional e schema da Transactional Outbox.
3. Schema físico/backend/TTL dos artifacts de snapshot.
4. Autenticação relay-to-relay inicial (mTLS, token ou NIP-98).
5. Scopes/filtros suportados na primeira release.
6. Limites/defaults após benchmarks.
7. Migration, índices e operação do replication slot.
8. Ativação no listener externo e anúncio NIP-11.
