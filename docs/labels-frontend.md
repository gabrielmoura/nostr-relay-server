# Frontend de Labels no `infra/dash`

## Objetivo

Este documento transforma a analise funcional da experiencia de labels da referencia em requisitos de frontend para o painel interno deste repositorio.

O alvo agora e a SPA administrativa em `infra/dash`, nao a antiga aplicacao `src/...` da referencia.

## Requisitos funcionais

### 1. Rota dedicada

- adicionar rota `/labels` em `infra/dash/src/router.tsx`;
- adicionar entrada de navegacao no `AppShell`;
- posicionar a tela junto do fluxo de moderacao, proxima de `/events/reported` e `/nip86`.

### 2. Fonte de dados server-side

A UI nao deve abrir WebSocket Nostr diretamente no browser para essa tela.

Ela deve usar apenas a camada tipada `services/admin.ts` para consumir:

- `GET /admin/labels`
- `GET /admin/labels/summary`
- `POST /admin/labels`

Motivos:

- alinhamento com o painel atual;
- cache consistente via TanStack Query;
- propagacao de `x-request-id` em erros;
- publicacao assinada no backend.

### 3. Estados visuais obrigatorios

- `loading`: skeletons para KPIs, filtros e listas;
- `error`: painel inline com retry e `requestId` quando disponivel;
- `empty`: estado vazio distinto para "sem labels" e "sem resultados para o filtro";
- `success`: workspace completo com contadores, filtros e duas visoes.

### 4. Duas visoes principais

#### Timeline

Cada item representa um evento `kind:1985` e deve mostrar:

- labels em badges;
- namespace;
- tipo de alvo;
- valor do alvo;
- relay hint, quando existir;
- autor do evento;
- data;
- comentario (`content`), quando existir.

#### By Target

Agrupar por `target.type + target.value`, mostrando:

- tipo do alvo;
- valor do alvo;
- quantidade de label events;
- labels deduplicados;
- ultimo label aplicado;
- acao rapida de ban quando o alvo for pubkey.

### 5. Filtros

Filtros minimos:

- namespace;
- um ou multiplos labels simultaneos;
- tipo de alvo;
- busca textual por alvo/comentario/autor;
- opcionalmente autor do label.

Os filtros devem ser refletidos em query params para permitir compartilhamento da URL.

### 6. Criacao de label

O dialog de criacao precisa suportar:

- alvo `event`, `pubkey`, `address`, `reference` e `topic`;
- aceitar NIP-19 em `Valor do alvo` quando aplicavel (`note`, `nevent`, `npub`, `nprofile`, `naddr`);
- quando `Tipo de alvo = pubkey`, `Valor do alvo` deve aceitar `hex`, `npub` e `nprofile`, permitindo labeling direto de um perfil/identidade;
- relay hint opcional para `event` e `pubkey`;
- namespace predefinido (`ugc`, `content-warning`, `dtsp`, `legal`, `moderation/resolution`) e custom;
- categorias predefinidas mais label customizado;
- comentario opcional;
- checkbox `tambem banir pubkey` quando o alvo for `pubkey`.

### 6.1. Ajuda contextual

A rota `/labels` deve expor um botao de ajuda que abre modal explicando:

- Tipo de alvo
- Namespace
- Valor do alvo
- Labels
- Comentario

### 7. Fluxo combinado de ban

Quando o operador marcar `tambem banir pubkey`:

1. a UI chama `POST /admin/labels`;
2. em sucesso, chama `POST /admin/users/:pubkey/ban` com motivo derivado dos labels;
3. invalida caches de labels, bans, perfil do usuario e overview.

### 8. Categorias base

O frontend deve oferecer categorias rapidas inspiradas na referencia, sem acoplar os nomes da UI a componentes copiados:

- `csam`
- `terrorism`
- `credible_threats`
- `doxxing`
- `malware`
- `nonconsensual`
- `hate`
- `illegal_goods`
- `violence`
- `self_harm`
- `nsfw`
- `spam`
- `impersonation`
- `copyright`

Tambem deve aceitar labels arbitrarios em lowercase.

## Decisoes de UX

Direcao visual aplicada ao painel atual:

- manter o estilo operacional existente do dashboard;
- usar o principio `data-dense + drill-down` da skill `ui-ux-pro-max` apenas como referencia estrutural;
- preservar a paleta atual do painel e evitar a direcao roxa generica sugerida pela busca automatica;
- destacar severidade com badges e tons semanticamente consistentes (`danger`, `warning`, `muted`, etc.);
- usar tipografia atual do app, com monoespaco apenas para ids, `note/nevent`, namespaces e targets tecnicos.

## Reuso previsto

O formulario de criacao deve ser projetado para reuso posterior em:

- `/events/reported`;
- `/events/$eventId`;
- `/users/$pubkey`.

No primeiro rollout, apenas a rota `/labels` e obrigatoria.

## Outras extensoes de UX relacionadas

- `/download` e `/sync` devem permitir limpar historico visivel de jobs;
- `/events/reported` deve mostrar cards KPI relevantes antes da lista;
- `/users/search` deve mostrar cards KPI relevantes para o resultado atual;
- `/events/$eventId` deve exibir tambem labels, reports, respostas, autores relacionados e eventos associados quando disponiveis.

## Pendencias abertas apos rollout inicial

- `NostrFilterBuilder` ainda deve ser ampliado para aceitar entrada NIP-19 ou hex nos campos relevantes;
- `/sync` apresenta bug em que um item cancelado pode voltar a executar; o comportamento desejado e estado terminal apos cancelamento;
- `/sync` tambem deve expor acao explicita de `retomar` para itens cancelados.

## Nao copiar da referencia

Pode reaproveitar a logica funcional analisada em `ref/divine-relay-manager`, mas nao copiar:

- componentes React;
- estrutura de arquivos da UI antiga;
- textos, classes ou composicao literal dos cards/dialogs.

O objetivo e uma implementacao nativa do `infra/dash`, aderente ao padrao atual do repositorio.
