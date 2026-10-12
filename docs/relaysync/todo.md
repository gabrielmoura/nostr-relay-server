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
- [ ] Implementar Negentropy V1 independentemente a partir da especificação.
- [ ] Usar packages existentes apenas como referência comportamental.
- [ ] Provar que nenhum código/dependência externa foi incorporado.
- [ ] Importar vetores/harness oficial de conformidade.
- [ ] Testar conjuntos iguais, parciais, vazios e grandes.
- [ ] Testar frames/versões/rounds inválidos e cancelamento.
- [ ] Fuzzar parser/codec.
- [ ] Provar limite de memória por partição.
- [ ] Testar snapshot estável durante writes/deletes.
- [ ] Testar expiração e liberação de recursos.

## Fase 3 — HTTP e zstd

- [ ] Implementar capabilities.
- [ ] Implementar sessions.
- [ ] Implementar manifest pagination.
- [ ] Implementar reconcile.
- [ ] Implementar events pull/push.
- [ ] Implementar complete/delete.
- [ ] Implementar cliente com `http.Client` injetável/reutilizado.
- [ ] Propagar deadlines/cancelamento.
- [ ] Implementar idempotency e typed errors.
- [ ] Implementar zstd negociado, threshold e output limits.
- [ ] Testar decompression bomb e corpos malformados.
- [ ] Testar sync HTTP E2E entre stores de teste.
- [ ] Benchmark sem compressão, nível 1 e nível 3.

## Fase 4 — journal e live

- [ ] Implementar journal catch-up `changes:next` para modo HTTP-only.
- [ ] Implementar cabeçalho WS binário de 8 bytes.
- [ ] Implementar hello/subprotocol/compression negotiation.
- [ ] Implementar subscribe e replay após watermark.
- [ ] Implementar DeltaBatch e JournalChange.
- [ ] Implementar ACK cumulativo contíguo.
- [ ] Implementar FlowControl e fila bounded.
- [ ] Implementar heartbeat/timeouts.
- [ ] Implementar ReconcileRequired para gap/epoch/scope.
- [ ] Implementar reconexão/checkpoint.
- [ ] Testar queda antes/depois de ACK.
- [ ] Testar rejeição permanente e falha temporária.
- [ ] Testar sync bidirecional e echo suppression.
- [ ] Testar race e goroutine leaks.

## Fase 5 — adapters genéricos e docs

- [ ] Finalizar adapter `net/http`.
- [ ] Implementar adapter Fiber 2.52.12 no listener fornecido.
- [ ] Manter TLS/HTTP2 sob responsabilidade do roteador/proxy.
- [ ] Criar suíte de conformidade dos ports.
- [ ] Criar exemplo net/http compilável.
- [ ] Criar exemplo Fiber compilável.
- [ ] Criar exemplo storage em memória compilável.
- [ ] Escrever README/interfaces/protocolo/segurança/performance/erros.
- [ ] Documentar garantias e limitações honestas.

## Fase 6 — NRServer

- [ ] Reauditar o código antes da integração.
- [ ] Criar adapter de conversão Nostr explícito.
- [ ] Criar snapshot materializado preservando payloads no TTL.
- [ ] Não manter transação entre requests.
- [ ] Criar facade síncrona de policy/persistência pós-commit.
- [ ] Registrar `/sync/v2/*` antes do fallback `/`.
- [ ] Adicionar configuração Viper/YAML, defaults false.
- [ ] Integrar auth por peer/scope.
- [ ] Integrar métricas/logs/readiness sem alta cardinalidade.
- [ ] Aprovar ADR CDC versus outbox.
- [ ] Aprovar migration/schema/índices/rollback.
- [ ] Implementar journal cobrindo todas as mutações.
- [ ] Testar restart, retention, gap, epoch e commit order.
- [ ] Implementar checkpoint persistente.
- [ ] Manter live desabilitado até gates completos.
- [ ] Testar coexistência/regressão NIP-01 e NIP-77.
- [ ] Anunciar extensão NIP-11 somente quando disponível.

## Fase 7 — performance e release

- [ ] Benchmark 100 mil eventos.
- [ ] Benchmark 1 milhão de eventos.
- [ ] Benchmark 10 milhões sob demanda.
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
- CDC/outbox, migration e artifact store antes da integração de journal.
- Mecanismo de autenticação e scopes antes de exportar dados reais.
- Resultados de benchmark antes de afirmar escala/produção.
