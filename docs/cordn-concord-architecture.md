# Cordn e Concord no NRServer

## Decisão

Cordn e Concord serão implementados no mesmo binário do NRServer, atrás de
feature flags independentes e desligadas por padrão. Eles serão domínios de
protocolo separados do Nostr relay comum e do RelaySync.

RelaySync permanece inalterado e continua responsável exclusivamente pela
replicação privada entre relays. Nenhum tipo, sessão, cursor, token,
`operation_id` ou armazenamento de RelaySync será reutilizado por Cordn ou
Concord.

## Visão do processo único

```text
nrserver server
├── Nostr relay (NIP-01/NIP-42 e NIPs existentes)
├── Cordn coordinator [cordn.enabled]
│   ├── transporte ContextVM sobre Nostr
│   ├── métodos Cordn/CEP-41
│   ├── storage de KeyPackages, Welcomes e Join Requests
│   └── mensagens MLS opacas com cursor por gid
├── Concord relay policy [concord.enabled]
│   ├── armazenamento/entrega de giftwraps Nostr
│   ├── visibilidade consistente em REQ, COUNT e live
│   └── sem decriptação ou autoridade sobre comunidades
└── RelaySync [relay_sync.*]
    └── protocolo privado de replicação, sem dependência dos dois domínios
```

O bootstrap cria somente os módulos cujas flags estão ativas. Com ambas as
flags desligadas, o caminho de inicialização e o comportamento atual do relay
permanecem inalterados.

## Cordn coordinator embutido

O coordinator é um serviço interno do processo Go, exposto por eventos Nostr
ContextVM plaintext (`kind:25910`) dirigidos à chave pública do relay. A
implementação não transforma o
relay em autoridade MLS: o servidor apenas autentica o chamador, valida os
limites e armazena/entrega os objetos que a especificação define como opacos.

### Superfície obrigatória

- publicação, listagem, consumo e remoção de KeyPackages;
- armazenamento e leitura de Welcomes pendentes;
- armazenamento e leitura/consumo de Join Requests;
- publicação e fetch de mensagens MLS opacas;
- subscription com replay do backlog seguido de entrega live;
- identidade do chamador injetada pelo transporte e vinculada às operações
  autorizadas;
- limites, rate limiting, expiração e encerramento de streams.

### Estado e durabilidade

O storage Cordn é próprio, com tabelas e queries separadas do `event` e do
RelaySync. Mensagens terão cursor monotônico por `gid`; não haverá cursor
global. O cursor só avança depois de a gravação durável ser confirmada.

Na transição `fetch → subscribe`, o coordinator registra a assinatura antes
de replayar o backlog. Mensagens que chegarem durante o replay ficam na fila
da assinatura. A entrega usa `(gid, cursor)` como chave de deduplicação na
fronteira, preservando a semântica da especificação sem interpretar o
payload MLS.

### Limite atual do transporte

O modo inicial cobre o transporte plaintext padrão do cliente Cordn. Envelopes
ContextVM criptografados por NIP-59 (`kind:21059`) não são aceitos pelo
coordinator: o relay não faz unwrap NIP-59/NIP-44 e não deve fingir que
consegue preservar a confidencialidade desse envelope. CEP-41 completo e a
interoperabilidade byte-a-byte com o SDK de referência permanecem uma etapa de
validação explícita antes de uso em produção.

### Segurança

O transporte deve validar a assinatura Nostr da requisição, injetar a pubkey
do cliente e impedir que NIP-42 seja tratado como substituto da identidade
ContextVM. A publicação de KeyPackage deve conferir a identidade do
BasicCredential com o signer da publicação. Tokens e material MLS nunca serão
registrados em logs.

## Concord no relay

Concord será implementado como compatibilidade de relay, não como serviço de
autoridade comunitária. O relay armazena e entrega eventos Concord opacos,
incluindo giftwraps, sem decifrar conteúdo, verificar membership ou calcular
roles, rekeys e refoundings.

A flag `concord.enabled` habilita uma política de leitura específica. Essa
política deve ser aplicada uniformemente a:

- backlog de `REQ`;
- `COUNT`;
- notificações live;
- caches de consulta;
- exportações e superfícies administrativas que possam expor conteúdo.

Adicionar `kind:1059` a uma lista genérica de kinds protegidos não é
suficiente: a regra precisa considerar o escopo do filtro e a identidade da
conexão, sem permitir descoberta ampla de giftwraps privados. O caminho de
ingestão continua usando a validação Nostr existente, com limites e rate limit
específicos para reduzir spam e correlação de metadados.

## Configuração

```yaml
cordn:
  enabled: false
  max_message_bytes: 262144
  max_pending_join_requests: 100
  retention_days: 30
  max_concurrent_subscriptions: 1000

concord:
  enabled: false
  # política de visibilidade e limites serão definidos pela implementação
```

As chaves são opcionais. Ausência equivale a `false`; configuração antiga não
deve iniciar nenhum módulo novo.

## Estado de implementação

1. Configuração tipada, migration e coordinator Cordn já são carregados pelo
   ciclo de vida do `server` apenas com `cordn.enabled: true`.
2. O handler Cordn roda após a validação normal de assinatura e política Nostr,
   mas antes da ingestão; pedidos e respostas são efêmeros e não entram em
   RelaySync nem na tabela de eventos.
3. A política Concord, ativada por `concord.enabled`, só permite consulta de
   `kind:1059` em filtro isolado com exatamente um autor de stream, cobrindo
   REQ, COUNT e a assinatura live criada a partir desses filtros.
4. A limpeza horária aplica `cordn.retention_days` a Welcomes, Join Requests e
   mensagens de grupo; KeyPackages seguem o seu próprio ciclo de consumo ou
   remoção. Ainda faltam validação externa contra SDK/fixtures Cordn, limites
   de taxa e suporte opcional a envelopes criptografados.

Casos de crash, replay duplicado, cancelamento de stream, gap de retenção e
vazamento de giftwrap devem ser validados antes de habilitar qualquer flag em
produção.
