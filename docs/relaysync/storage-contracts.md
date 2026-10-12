# RelaySync v2: contratos de storage

Status: contrato planejado da Fase 0. As assinaturas podem receber ajustes
compatíveis durante a Fase 1, antes da primeira versão pública.

## Princípios

- Interfaces representam operações do domínio, não SQL genérico.
- Nenhuma operação retorna milhões de itens em um slice.
- Todo I/O recebe `context.Context` e respeita cancelamento/deadline.
- Iteradores têm `Close` idempotente e expõem erro terminal.
- IDs, cursores e versões são tipos fortes.
- O host mantém políticas de visibilidade, retenção e aceitação.
- Uma implementação só é compatível depois de passar a suíte de conformidade.

## Tipos base

```go
type SnapshotVersion []byte

type Cursor struct {
    Epoch    uint64
    Sequence uint64
}

type Scope struct {
    Name    string
    Version uint32
    Hash    [32]byte
}

type EventMetadata struct {
    ID        EventID
    CreatedAt int64
}
```

`Scope.Hash` identifica a forma canônica das regras negociadas, não uma
serialização Protobuf arbitrária. Cursores são comparáveis somente dentro da
mesma epoch.

## SnapshotProvider

Forma inicial do port:

```go
type SnapshotProvider interface {
    OpenSnapshot(ctx context.Context, scope Scope) (Snapshot, error)
}

type Snapshot interface {
    Descriptor() SnapshotDescriptor
    OpenMetadata(ctx context.Context, partition PartitionKey,
        after *MetadataCursor) (MetadataIterator, error)
    GetEvents(ctx context.Context, ids []EventID) (EventIterator, error)
    Close() error
}
```

`SnapshotDescriptor` contém versão opaca, scope, watermark, instante de
expiração e plano de partições. Todas as chamadas da mesma instância devem ver
o mesmo conjunto, inclusive inserts, substituições, deleções e expirações que
ocorram depois da abertura.

Garantias obrigatórias:

1. `Version` identifica de forma imutável a visão durante o TTL.
2. `Watermark` é um head já durável do journal. Qualquer mudança durável ausente
   do snapshot aparece depois dele; uma mudança presente pode reaparecer depois
   dele e deve ser aplicada idempotentemente.
3. Metadados são ordenados por `(created_at, event_id)` e paginados por cursor
   exclusivo, nunca por offset instável.
4. `GetEvents` só retorna eventos visíveis naquela versão e preserva a relação
   ID/conteúdo. Itens ausentes são explicitados, não omitidos silenciosamente.
5. Expiração retorna `ErrSnapshotExpired`; o caller deve reiniciar sessão.
6. `Close` libera pins, handles e caches mesmo após cancelamento.

Implementações conformes materializam a visão com TTL ou usam storage
versionado. Manter transação de banco aberta entre requisições não é conforme,
pois ocupa conexão, prejudica vacuum e tem duração imprevisível.

## EventWriter

```go
type EventWriter interface {
    WriteEvents(ctx context.Context, batch EventBatch) ([]WriteResult, error)
}

type WriteStatus uint8

const (
    WriteStatusUnspecified WriteStatus = iota
    WriteAccepted
    WriteAlreadyExists
    WriteRejectedPermanent
    WriteFailedTemporary
)
```

Cada `WriteResult` contém Event ID, status, código tipado e mensagem segura. A
ordem corresponde à entrada. Um erro global indica que o resultado do batch é
desconhecido; o caller deve consultar/repetir idempotentemente.

Garantias obrigatórias:

- `OperationID` identifica retry do mesmo batch.
- evento aceito está duravelmente publicado antes de retornar `Accepted`;
- duplicata equivalente retorna `AlreadyExists`;
- rejeição permanente é determinística para a política/versão informada e pode
  ser gravada como decisão terminal;
- falha temporária nunca é convertida em rejeição terminal;
- assinatura, ID NIP-01, autorização e política do host são aplicados antes da
  aceitação;
- o adapter do NRServer reutiliza sua persistência/políticas; não cria tabela
  paralela de eventos.

Resultados parciais são permitidos, mas cada item deve ter resultado. O caller
só confirma um cursor quando todas as mudanças até ele têm resultado durável
terminal.

## JournalProvider

```go
type JournalProvider interface {
    OpenJournal(ctx context.Context, scope Scope, after Cursor) (Journal, error)
}

type Journal interface {
    Epoch() uint64
    Next(ctx context.Context) (JournalChange, error)
    Close() error
}
```

`JournalChange` contém cursor, tipo inicial (`UPSERT` ou `DELETE`), Event ID,
metadados mínimos e, quando autorizado, evento. Expiração, replacement e NIP-09
são projetados como mudança de visibilidade `DELETE`. Rejeições de entrada
pertencem ao store de decisão/idempotência, não ao journal de eventos. Tipos
desconhecidos não podem ser ignorados silenciosamente.

Garantias obrigatórias:

1. Sequências são estritamente crescentes e contíguas dentro da epoch.
2. A ordem reflete publicação durável/commit, não a ordem de chamada de
   `nextval()`.
3. O journal sobrevive a reinício e pode repetir o último item; consumers são
   idempotentes.
4. Cursor anterior à retenção retorna `ErrJournalGap` com o primeiro cursor
   disponível e exige nova reconciliação.
5. Mudança de epoch retorna `ErrJournalEpochChanged` e exige reconciliação.
6. O mesmo scope usado pelo snapshot é aplicado ao journal.
7. Redis Pub/Sub pode notificar disponibilidade, mas nunca é a fonte durável.

### Alternativas para o NRServer

| Alternativa | Ordem de commit | Custo/risco | Decisão da Fase 0 |
| --- | --- | --- | --- |
| `BIGSERIAL/nextval()` isolado | Não | Simples, mas incorreto com commits fora de ordem | Rejeitada |
| Redis Pub/Sub/Streams isolado | Não cobre transação PostgreSQL | Lacunas e dual-write | Rejeitada |
| Logical decoding/CDC | Sim | `wal_level`, slot, retenção de WAL e operação dedicada | Recomendada, sujeita a ADR/gate operacional |
| Outbox na mesma transação com sequenciamento serializado até commit | Sim, se todos os writers participarem | Refatorar todos os writers, contenção e migration | Alternativa para benchmark |

A recomendação inicial para o NRServer é CDC porque hoje existem múltiplos
caminhos de insert, replace, soft delete e hard delete. Um projector consome
commits em ordem de LSN, atribui uma sequência densa ao journal persistido e só
então confirma o replication slot. O LSN é checkpoint interno, não cursor do
protocolo. A Fase 6 deve medir essa opção e comparar com outbox serializada.
Live sync fica desabilitado até que retenção, restart, gap e todos os caminhos
de mutação passem nos testes. Não será inferida uma garantia a partir do
`BIGSERIAL` atual.

## CheckpointStore

```go
type CheckpointStore interface {
    LoadCheckpoint(ctx context.Context, key CheckpointKey) (Checkpoint, bool, error)
    AdvanceCheckpoint(ctx context.Context, next Checkpoint) error
}
```

`CheckpointKey` combina uma identidade estável de peer com scope. IP não é
identidade. `AdvanceCheckpoint` é compare-and-advance atômico: mesma epoch não
pode regredir sequência; epoch diferente exige reset explicitamente autorizado
após reconciliação.

O checkpoint só avança depois que eventos ou decisões terminais estão duráveis.
Rejeições permanentes carregam um `DecisionID` estável para evitar loops.
Falhas temporárias não avançam o cursor.

## SessionStore

O contrato será segregado para permitir implementações mínimas:

```go
type SessionCreator interface {
    CreateSession(ctx context.Context, session SessionRecord) error
}

type SessionReader interface {
    GetSession(ctx context.Context, id SessionID) (SessionRecord, error)
}

type SessionUpdater interface {
    UpdateSession(ctx context.Context, update SessionUpdate) error
}

type SessionCloser interface {
    DeleteSession(ctx context.Context, id SessionID) error
}
```

O record liga sessão, hash do token, peer, scope, snapshot version, estado,
watermark e expiração. Updates usam versão/CAS para impedir duas requisições de
avançarem a mesma máquina de estados.

A implementação em memória:

- usa mutex por store e estado imutável/copiado;
- expira por acesso e sweeper limitado;
- aceita `Clock` para testes;
- não é anunciada como distribuída.

## Autorização

```go
type PeerAuthorizer interface {
    Authorize(ctx context.Context, principal Principal,
        action Action, requested Scope) (AuthorizedScope, error)
}
```

A autenticação produz `Principal`; a autorização decide capabilities privadas,
abertura de sessão, pull, push e live separadamente. Um scope autorizado pode
ser mais restrito que o solicitado. O engine nunca assume acesso global.

## Conformidade de adapters

`pkg/relaySync/conformance` receberá factories e executará, no mínimo:

- snapshot estável diante de gravação concorrente;
- paginação sem perda/duplicação e ordem total;
- expiração e liberação de recursos;
- resultados individuais e retry idempotente de escrita;
- commits concorrentes fora da ordem de alocação de IDs;
- replay, duplicação tolerada, gap, retenção e epoch;
- checkpoint monotônico sob concorrência;
- CAS e expiração de sessão;
- cancelamento/deadline em todas as operações.

O adapter PostgreSQL será testado contra uma instância real; mocks servem apenas
para falhas unitárias, não para demonstrar isolamento ou ordem de commit.

## Referências

- [PostgreSQL: sequence manipulation](https://www.postgresql.org/docs/current/functions-sequence.html)
- [PostgreSQL: logical decoding concepts](https://www.postgresql.org/docs/current/logicaldecoding-explanation.html)
- [NIP-09](https://github.com/nostr-protocol/nips/blob/master/09.md)
- [NIP-40](https://github.com/nostr-protocol/nips/blob/master/40.md)
