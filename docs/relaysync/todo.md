# RelaySync v2: checklist

Legenda: `[x]` concluído e verificado; `[ ]` pendente; `[-]` não aplicável ou
adiado com justificativa. Este checklist não transforma plano em capacidade.

## Fase 0 — auditoria

- [x] Ler `.codex/config.toml`, `AGENTS.md` e agents do projeto.
- [x] Verificar `go.mod`, dependências, Makefile/scripts e CI.
- [x] Verificar `gopls`, `protoc`, `protoc-gen-go`, staticcheck e Buf.
- [x] Mapear servidor Fiber, rotas e lifecycle WebSocket.
- [x] Mapear NIP-77/Negentropy atual.
- [x] Mapear validação/policies e ingestão de eventos.
- [x] Mapear PostgreSQL, Redis, concorrência e transações.
- [x] Mapear autenticação, configuração, métricas e logging.
- [x] Mapear testes/benchmarks existentes.
- [x] Consultar especificações obrigatórias.
- [x] Levantar Vector, BTreeMem, BTreeLMDB, SubRange e HORNET go-negentropy.
- [x] Registrar path exato `pkg/relaySync` e `package relaysync`.
- [x] Documentar arquitetura, protocolo, storage, coexistência e plano.
- [x] Obter aprovação explícita da documentação.

## Fase 1 — core e contracts

- [x] Criar `pkg/relaySync` sem `go.mod` aninhado.
- [x] Criar `EventID`, `PublicKey`, `Signature`, `Tag` e `Event`.
- [x] Implementar ID canônico NIP-01 e testes/vetores.
- [x] Definir errors/codes tipados.
- [x] Definir options, limites finitos e validação.
- [x] Definir SnapshotProvider/Snapshot/iterators.
- [x] Definir EventWriter e resultados individuais.
- [x] Definir JournalProvider e cursor epoch/sequence opaco no wire.
- [x] Definir CheckpointStore monotônico.
- [x] Definir SessionStore segregado e CAS.
- [ ] Definir PeerAuthorizer e observabilidade opcional.
- [x] Implementar stores em memória.
- [x] Criar `relaysync.proto` proto3.
- [x] Fixar runtime/tool Protobuf.
- [x] Criar target reproduzível de geração.
- [x] Criar descriptor/golden compatibility tests.
- [x] Testar roundtrip, oneof, enums e tamanhos.
- [x] Executar test/race/vet focados (gopls bloqueado pelo cache somente leitura).

## Fase 2 — snapshot, manifest e Negentropy

- [x] Implementar partição 30 dias + prefixo/shard determinístico.
- [x] Implementar digest canônico independente de Protobuf.
- [ ] Implementar manifest paginado e cache versionado.
- [ ] Implementar Vector selado por partição, SubRange lógico e prefix aggregates.
- [ ] Criar porta privada de reconciliação.
- [x] Implementar Negentropy V1 independentemente a partir da especificação.
- [x] Usar packages existentes apenas como referência comportamental.
- [x] Provar que nenhum código/dependência externa foi incorporado.
- [ ] Importar vetores/harness oficial de conformidade.
- [ ] Testar conjuntos iguais, parciais, vazios e grandes.
- [ ] Testar frames/versões/rounds inválidos e cancelamento.
- [ ] Fuzzar parser/codec.
- [ ] Provar limite de memória por partição.
- [ ] Testar snapshot estável durante writes/deletes.
- [ ] Testar expiração e liberação de recursos.

## Fase 3 — HTTP e zstd

- [x] Implementar capabilities.
- [x] Implementar sessions.
- [x] Implementar manifest pagination.
- [x] Implementar reconcile HTTP orchestration stateless e bounded.
- [x] Implementar events pull/push.
- [x] Implementar complete/delete e descarte de ciclo.
- [x] Implementar cliente com `http.Client` injetável/reutilizado.
- [ ] Propagar deadlines/cancelamento.
- [ ] Implementar idempotency e typed errors.
- [x] Implementar zstd negociado, threshold e output limits.
- [ ] Testar decompression bomb e corpos malformados.
- [ ] Testar sync HTTP E2E entre stores de teste.
- [ ] Benchmark sem compressão, nível 1 e nível 3.

## Fase 4 — journal e live

- [ ] Implementar journal catch-up `changes:next` para modo HTTP-only.
- [x] Implementar adapter WebSocket binário e subprotocolo `nostr-sync.v2.pb`.
- [x] Implementar hello, flow-control, replay, delta e ACK no adapter Fiber/net/http.
- [x] Implementar DeltaBatch e JournalChange no core live.
- [x] Implementar ACK cumulativo contíguo.
- [x] Implementar FlowControl e fila bounded.
- [x] Implementar heartbeat/timeouts.
- [x] Implementar ReconcileRequired para gap/epoch/scope.
- [x] Implementar tabela e `CheckpointStore` PostgreSQL persistente para reconexão.
- [ ] Testar queda antes/depois de ACK.
- [ ] Testar rejeição permanente e falha temporária.
- [ ] Testar sync bidirecional e echo suppression.
- [ ] Testar race e goroutine leaks.

## Fase 5 — adapters genéricos e docs

- [x] Finalizar adapter `net/http`.
- [x] Implementar adapter Fiber 2.52.12 no listener fornecido.
- [x] Manter TLS/HTTP2 sob responsabilidade do roteador/proxy.
- [ ] Criar suíte de conformidade dos ports.
- [ ] Criar exemplo net/http compilável.
- [ ] Criar exemplo Fiber compilável.
- [ ] Criar exemplo storage em memória compilável.
- [ ] Escrever README/interfaces/protocolo/segurança/performance/erros.
- [ ] Documentar garantias e limitações honestas.

## Fase 6 — NRServer

- [ ] Reauditar o código antes da integração.
- [ ] Criar adapter de conversão Nostr explícito.
- [x] Criar snapshot PostgreSQL paginado com TTL e descarte de payloads ao fim do ciclo.
- [ ] Não manter transação entre requests.
- [x] Criar adapter EventWriter síncrono sobre o writer PostgreSQL existente;
  policy final permanece injetada pelo host.
- [x] Registrar `/sync/v2/*` antes do fallback `/`.
- [ ] Adicionar configuração Viper/YAML, defaults false.
- [x] Integrar allowlist por IP/escopo sem autenticação externa.
- [ ] Integrar métricas/logs/readiness sem alta cardinalidade.
- [x] Decidir Transactional Outbox PostgreSQL; Redis Pub/Sub somente notificação.
- [x] Criar migration aditiva de outbox/journal e trigger transacional.
- [x] Criar projector bounded com ordenação por commit e retenção separada.
- [ ] Validar migration/schema/índices/rollback em PostgreSQL real.
- [x] Implementar descarte de sessão/snapshot/idempotência ao completar ciclo.
- [ ] Validar ciclo finito ponta a ponta com retry e expiração.
- [x] Implementar leitura streaming do journal PostgreSQL por cursor/escopo.
- [ ] Completar cobertura do journal para todas as mutações em produção.
- [x] Integrar projector ao lifecycle supervisionado do NRServer (desabilitado por default).
- [ ] Testar restart, retention, gap, epoch e commit order.
- [x] Implementar checkpoint persistente.
- [ ] Manter live desabilitado até gates completos.
- [ ] Testar coexistência/regressão NIP-01 e NIP-77.
- [x] Anunciar extensão NIP-11 somente quando HTTP RelaySync estiver habilitado.

## Fase 7 — performance e release

- [x] Benchmark 100 mil eventos (digest e zstd).
- [x] Benchmark 1 milhão de eventos (digest).
- [ ] Benchmark 10 milhões sob demanda (`RELAYSYNC_BENCH_LARGE=1`).
- [ ] Medir divergências 0%, 0,1%, 1%, 10%, 50%, 100%.
- [ ] Coletar CPU, RSS, heap, allocs e GC.
- [ ] Coletar throughput/bytes/rounds/requests.
- [ ] Coletar live lag/recovery/backpressure.
- [ ] Executar pprof/trace e otimizar apenas gargalos medidos.
- [ ] Executar full test/race/static analysis/integration.
- [ ] Revisão de segurança e riscos operacionais.
- [ ] Tutorial de uso e configuração.
- [ ] Separar concluído, experimental e pendente na entrega.

## Bloqueios/gates atuais

- Aprovação da documentação antes de código.
- Conformidade da implementação Negentropy independente antes da Fase 2.
- Transactional Outbox, migration e artifact store antes da integração de journal.
- Mecanismo de autenticação e scopes antes de exportar dados reais.
- Resultados de benchmark antes de afirmar escala/produção.
