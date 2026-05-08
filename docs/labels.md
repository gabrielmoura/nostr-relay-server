# Labels NIP-32 no Relay

## Objetivo

Este documento consolida a analise de `docs/labels-frontend.md`, do NIP-32 via MCP Nostr, e da implementacao de referencia em `ref/divine-relay-manager` para definir como o `nostr-relay-server` deve expor gerenciamento de labels no painel interno `infra/dash`.

Ele substitui a antiga descricao de uma UI React externa (`src/...`) que nao pertence a este repositorio.

## Leitura do NIP-32

Com base no MCP Nostr (`NIP-32`):

- labels sao eventos `kind:1985`;
- `L` define o namespace;
- `l` define o valor do label e deve apontar para o namespace quando `L` existir;
- o alvo pode ser `e`, `p`, `a`, `r` ou `t`;
- `content` guarda justificativa longa;
- `e` e `p` podem carregar relay hint na terceira posicao;
- labels podem coexistir com outros fluxos de moderacao, inclusive resolucao operacional.

## O que ja existe no relay

- o banco ja armazena eventos `kind:1985` na tabela `event`;
- o schema atual usa `tags JSONB`, o que permite filtros por `L`, `l`, `e`, `p`, `a`, `r` e `t` sem criar nova tabela;
- consultas administrativas de reports (`kind:1984`) ja usam `jsonb_array_elements(...)` em `infra/db/admin_query.go`, padrao que sera reaproveitado para labels;
- a configuracao ja possui `relay_information.priv_key`, permitindo que o backend assine labels administrativos com a identidade do relay.

## Referencia analisada

`ref/divine-relay-manager` foi usado apenas como referencia funcional.

Capacidades observadas e que fazem sentido reaproveitar no relay atual:

- listagem cronologica de `kind:1985`;
- agrupamento por alvo moderado;
- criacao de label com comentario opcional;
- opcao de banir pubkey apos rotular;
- uso de namespace `moderation/resolution` para workflow operacional;
- normalizacao visual de categorias frequentes (`spam`, `nsfw`, `csam`, `malware`, `impersonation`, etc.).

Capacidades que precisam ser adaptadas ao repositorio atual:

- a referencia usa browser + worker para publicar eventos assinados; aqui a assinatura deve acontecer no backend Go;
- a referencia trata principalmente `e` e `p`; aqui a implementacao alvo deve suportar tambem `a`, `r` e `t`;
- a referencia depende de componentes `src/...`; aqui a UI deve ser implementada dentro de `infra/dash` seguindo o padrao TanStack Router + TanStack Query.

## Escopo da feature no relay atual

### Backend

Adicionar uma superficie administrativa interna para labels em `/admin`:

- `GET /admin/labels`
- `GET /admin/labels/summary`
- `POST /admin/labels`

Responsabilidades:

- consultar labels persistidos no Postgres;
- expor filtros operacionais por namespace, um ou multiplos labels, alvo, autor e tipo de alvo;
- agregar contagens por namespace, label e alvo;
- criar novos eventos `kind:1985` assinados com `relay_information.priv_key`;
- persistir o evento no banco reaproveitando o modelo de evento Nostr ja existente;
- deixar o ban opcional como orquestracao do frontend usando o endpoint ja existente `POST /admin/users/:pubkey/ban`.

### Frontend

Adicionar uma nova rota do painel interno:

- `/labels`

Responsabilidades:

- exibir KPIs e filtros rapidos;
- mostrar timeline de label events;
- mostrar agregacao por alvo;
- abrir dialog de criacao de label;
- permitir label customizado, namespace customizado e comentario opcional;
- permitir fluxo combinado de `criar label + banir pubkey`;
- reaproveitar o mesmo formulario em pontos contextuais futuros, como reports e event detail.

## Modelo funcional alvo

### Targets suportados

O painel deve aceitar todos os alvos relevantes do NIP-32:

- `event` -> tag `e`
- `pubkey` -> tag `p`
- `address` -> tag `a`
- `reference` -> tag `r`
- `topic` -> tag `t`

Representacao administrativa unificada:

```json
{
  "type": "event | pubkey | address | reference | topic",
  "value": "...",
  "relay_hint": "opcional para e/p"
}
```

Para `type = pubkey`, o operador deve poder informar:

- hex canonico
- `npub`
- `nprofile`

Todos os formatos devem ser normalizados para hex antes da escrita do evento. Isso significa que sim, o fluxo suporta labeling direto de um perfil/identidade Nostr.

### Payload de criacao

```json
{
  "namespace": "ugc",
  "labels": ["spam", "scam"],
  "comment": "Conta usada para flood promocional.",
  "target": {
    "type": "pubkey",
    "value": "<hex-pubkey>",
    "relay_hint": "wss://relay.example"
  }
}
```

Evento produzido:

```json
{
  "kind": 1985,
  "content": "Conta usada para flood promocional.",
  "tags": [
    ["L", "ugc"],
    ["l", "spam", "ugc"],
    ["l", "scam", "ugc"],
    ["p", "<hex-pubkey>", "wss://relay.example"]
  ]
}
```

## Decisoes de implementacao propostas

### Persistencia

- nenhuma nova tabela sera criada para NIP-32;
- o armazenamento continua na tabela `event`;
- a consulta administrativa usa SQL dedicado com `jsonb_array_elements(tags)` para extrair namespace, labels e target;
- o filtro administrativo por labels deve aceitar repeticao de `label=` com semantica OR para permitir multi-selecao no dashboard;
- agregacoes operacionais tambem ficam no Postgres, evitando duplicacao em Redis.

### Publicacao

- o backend cria e assina o `kind:1985` usando a chave do relay;
- o primeiro rollout considera sucesso quando o evento esta validamente persistido no relay local;
- o fluxo nao depende de um worker externo nem de publicacao browser-side.

### Moderacao

- labels nao substituem bans nem reports;
- labels complementam `kind:1984` e o ban manual;
- o namespace `moderation/resolution` continua reservado para marcacoes operacionais futuras ou integracoes com a tela de reports.

## Riscos e trade-offs

- publicar labels pelo backend usando a chave do relay simplifica operacao, mas deixa explicito que o autor do label sera a identidade administrativa do relay, nao o navegador do operador;
- manter tudo na tabela `event` evita migracao, mas exige queries JSONB bem testadas para nao degradar o painel;
- suportar `a`, `r` e `t` no primeiro rollout aumenta cobertura do NIP-32 e evita repetir a limitacao da referencia, ao custo de um formulario um pouco mais rico.

## Fora de escopo deste rollout

- sincronizar labels com sistemas externos;
- criar taxonomia persistida em tabela propria;
- auditoria multi-ator com identidades separadas por moderador;
- automacao de enforcement com base em labels `kind:1985` sozinhos.
