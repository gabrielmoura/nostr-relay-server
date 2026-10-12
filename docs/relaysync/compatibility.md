# RelaySync v2: compatibilidade e coexistência

Status: proposta da Fase 0; decisões explícitas do usuário estão identificadas
nos documentos e o restante aguarda aprovação.

## Identidades distintas

| Aspecto | NIP-77 existente | RelaySync v2 |
| --- | --- | --- |
| Status | NIP draft/optional | protocolo privado experimental |
| Rota | WebSocket Nostr raiz | `/sync/v2/*` e `/sync/v2/live` |
| Mensagens | JSON textual `NEG-OPEN`, `NEG-MSG`, `NEG-CLOSE`, `NEG-ERR` | Protobuf binário |
| Negentropy | frame hex no JSON | frame V1 cru em `bytes` |
| Transferência | `EVENT`/`REQ` NIP-01 | pull/push em batches Protobuf |
| Sessão | estado próprio do handler atual | `SessionStore` RelaySync |
| Live journal | não definido pelo NIP-77 | cursor epoch+sequence e ACK |
| Feature flag | configuração Negentropy existente | flags RelaySync independentes |

O nome do novo diretório é `pkg/relaySync`. `pkg/negentropyV2` permanece o
package existente do NIP-77 e não será renomeado, removido ou transformado na
nova biblioteca.

## Regras de coexistência

1. Não alterar parsing, resposta, limites ou estado de `NEG-*`.
2. Não alterar mensagens NIP-01 (`EVENT`, `REQ`, `EOSE`, `OK`, `CLOSED`).
3. Não registrar `/sync/v2/live` no upgrade genérico da raiz.
4. Não compartilhar objetos de sessão, tokens ou namespaces de IDs.
5. Não usar cache RelaySync como cache implícito do NIP-77.
6. Compartilhar somente ports puros ou adapters de storage com semântica
   comprovadamente igual.
7. Habilitar NIP-77 e RelaySync separadamente.
8. Falha/overload no RelaySync não pode fechar sessões NIP-77.

## Registro de rotas

O router Fiber existente deve registrar `/sync/v2` antes da rota/catch-all que
serve NIP-11 e WebSocket Nostr, mantendo o listener atual:

```text
/sync/v2/capabilities       RelaySync HTTP
/sync/v2/sessions...        RelaySync HTTP
/sync/v2/live               RelaySync WebSocket binário
/                           NIP-11 ou WebSocket Nostr/NIP-77
```

O subprotocolo `nostr-sync.v2.pb` é obrigatório na rota live. Um cliente Nostr
comum que conecta em `/` não entra acidentalmente no RelaySync.

HTTP/2/TLS é responsabilidade do roteador/reverse proxy, conforme decisão do
projeto. O adapter Fiber não cria servidor ou certificado adicional.

## Configuração independente

Estrutura conceitual, a ser alinhada ao Viper/YAML real na Fase 6:

```yaml
relay_sync:
  enabled: false
  snapshot_enabled: false
  apply_enabled: false
  catchup_enabled: false
  live_enabled: false
```

`enabled` é o gate mestre. Snapshot/read, apply/write, catch-up HTTP e live têm
gates próprios; `live_enabled` exige `catchup_enabled`, journal, checkpoints e
snapshot/watermark conformes. O default de todos é `false`. As opções
Negentropy/NIP-77 existentes não são reutilizadas como gate.

## Componentes compartilháveis

Pode ser compartilhado por adapter, nunca por import do core para o host:

- leitura paginada de metadados PostgreSQL;
- busca de eventos por ID;
- validação NIP-01 e política de aceitação;
- persistência batch/COPY, se devolver resultado durável individual correto;
- infraestrutura Prometheus/logging por bridge de interfaces;
- autenticação NIP-42/NIP-98 ou mTLS para produzir `Principal`.

Não é seguro compartilhar diretamente:

- `dto.WsServer` e canais da conexão Nostr;
- manager/session maps de `pkg/negentropyV2`;
- globals de `db`, Redis, config ou logger;
- ACK de enqueue da ingestão normal;
- Redis Pub/Sub como journal;
- logging raw RX/TX existente.

## NIP-11

O número `77` em `supported_nips` continua anunciando somente NIP-77. Quando e
somente quando RelaySync estiver efetivamente disponível, o documento NIP-11
poderá incluir extensão privada fora de `supported_nips`, por exemplo:

```json
{
  "x_relaysync": {
    "versions": [2],
    "capabilities_url": "/sync/v2/capabilities",
    "live": false
  }
}
```

O nome final da extensão será revisado antes da integração. `live` nunca será
anunciado se o journal estiver desabilitado ou não saudável.

## Semântica de convergência

RelaySync não presume bancos idênticos. A convergência é declarada apenas para
a tupla:

```text
(scope name, scope version, policy hash, snapshot version)
```

Scopes excluem ou descrevem explicitamente:

- eventos efêmeros;
- eventos privados/restritos;
- retenção e expiração NIP-40;
- deleções NIP-09;
- regras para substituíveis/endereçáveis;
- kinds, autores e intervalos permitidos;
- política de rejeição local.

Se os hashes de política não são compatíveis, os peers podem sincronizar o
subconjunto negociado, mas não declarar convergência global. Rejeição
permanente é uma decisão terminal identificada; mudança posterior de política
invalida essa decisão e exige novo scope/versionamento ou reconciliação.

## Matriz de regressão

Antes da integração ser aceita:

| Cenário | Resultado esperado |
| --- | --- |
| apenas NIP-77 habilitado | comportamento e wire bytes existentes |
| apenas RelaySync HTTP habilitado | `/sync/v2/*` funcional; `NEG-*` não ativado por isso |
| ambos habilitados | sessões e limites independentes |
| RelaySync desabilitado | rotas ausentes/404 e NIP-11 sem extensão |
| live desabilitado | HTTP funciona; upgrade live recusado claramente |
| overload RelaySync | NIP-01/NIP-77 continuam atendidos dentro de seus limites |
| shutdown | ambos fecham seus próprios recursos sem goroutine leak |

Antes da integração será criado um baseline/golden do wire NIP-77 atual. Ele não
será regravado para acomodar o RelaySync; qualquer diferença será tratada como
regressão até prova em contrário.

## Autenticação

NIP-42 atual pode inspirar ou produzir identidade, mas seu estado de conexão
Nostr não é reutilizado automaticamente na rota live. HTTP pode usar mTLS,
NIP-98 ou middleware próprio. Todos resultam em um `Principal` estável e não em
IP. O token da sessão RelaySync é adicional, curto e vinculado a principal e
scope.

## Critério para deixar de ser experimental

O NRServer não anunciará RelaySync como pronto para produção até:

- conformidade Negentropy e Protobuf passar;
- snapshot/journal/checkpoint não perderem commits sob falhas testadas;
- autorização por scope estar integrada;
- reconexão, gaps e epochs passarem em integração real;
- race detector e testes de coexistência passarem;
- benchmarks de 100 mil e 1 milhão tiverem resultados documentados;
- o cenário de 10 milhões estiver medido ou explicitamente marcado como não
  validado.

## Referências

- [NIP-11](https://github.com/nostr-protocol/nips/blob/master/11.md)
- [NIP-42](https://github.com/nostr-protocol/nips/blob/master/42.md)
- [NIP-98](https://github.com/nostr-protocol/nips/blob/master/98.md)
- [NIP-77](https://github.com/nostr-protocol/nips/blob/master/77.md)
