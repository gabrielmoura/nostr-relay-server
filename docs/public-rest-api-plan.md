# Plano de implementação — API REST pública de leitura

**Status:** proposta, aguardando aprovação antes de qualquer alteração de código.

**Referência do contrato:** [nostrfy REST API](https://nostrfy.org/docs/rest-api/).

## 1. Objetivo e limites

Implementar uma API HTTP somente de leitura sob /api/v1/ na mesma porta externa
do relay (config.Cfg.Port). Ela replica o contrato do nostrfy usando Go, Fiber
e PostgreSQL já existentes; não porta código Rust nem altera o protocolo
WebSocket.

Ficam fora do escopo: escrita HTTP, alterações em Admin/GraphQL/painel, mudança
do comportamento de REQ WebSocket e a Fase 6 de roteamento por Host, salvo
aprovação específica.

## 2. Estado confirmado do repositório

| Área | Estado atual | Consequência |
| --- | --- | --- |
| Consulta | db.Queries.QueryEvents e CountEvents recebem nostr.Filter, usam SQL comum e cache Redis. | Construir filtros HTTP e reutilizar essas funções; não recriar consulta genérica. |
| NIP-19 | admin_search_helpers.go normaliza npub, note, nevent, naddr e tags. | Extrair helpers semânticos compartilhados, mantendo o admin consumidor deles. |
| Servidor externo | RouterFactory.setupExternalRoutes monta Fiber em Port; a app interna usa Port+1. | Registrar a API somente na aplicação externa. |
| Grupos | groups.QueryEvents e ValidateFilter aceitam authed vazio e filtram por grupo. | Reutilizar no fluxo anônimo, inclusive agregações. |
| NIP-70/NIP-59 | Leitura anônima atual não descarta tag [-] nem kind 1059. | Criar filtro de visibilidade de leitura sem mudar WebSocket. |
| Paginação | QueryEvents usa ordem DESC fixa e não recebe offset. | Criar paginação própria, estável e baseada na sequência visível. |
| Configuração | Não há bloco api, api_host ou NIP-98 da API. | Introduzir somente nas fases previstas, com defaults documentados. |

## 3. Contrato HTTP

### Regras comuns

- Somente GET é aceito sob /api/v1; outros métodos retornam 405 se alcançarem
  o roteador.
- Upgrade WebSocket sob /api/v1 retorna 403.
- Sucesso de consultas de eventos usa o envelope:

    {
      "events": [{"id":"...","pubkey":"...","created_at":0,"kind":1,"tags":[],"content":"","sig":"..."}],
      "count": 1,
      "more": false
    }

- count é o número de eventos desta página. O endpoint /count segue NIP-45
  sobre o conjunto visível.
- Erros seguem o formato JSON já adotado pelos handlers HTTP do repositório,
  identificado na Fase 5.0, sem vazar SQL ou detalhes internos.

### Rotas por identificador

| Rota | Filtro ou resultado | Erro específico |
| --- | --- | --- |
| GET /api/v1/:npub | Autor + kind 0, limitado ao perfil mais recente. | 400 para NIP-19 inválido. |
| GET /api/v1/:note ou /:nevent | ID exato, um evento. | 400 para NIP-19 inválido. |
| GET /api/v1/:naddr | kind, autor e tag d do ponteiro. | 400 para ponteiro inválido. |
| GET /api/v1/:npub/:kind | Autor + kind. | 400 se o primeiro segmento não for npub ou o kind for inválido. |

### Rotas de consulta e agregação

| Rota | Resultado esperado |
| --- | --- |
| /api/v1/query | Página de eventos pelo filtro genérico. |
| /api/v1/count | Contagem NIP-45 do filtro. |
| /api/v1/:npub/kinds | Contagens por kind do autor, maior uso primeiro. |
| /api/v1/:npub/:kind/daily | Dias de um mês, zero-filled até o último dia. |
| /api/v1/ids/:hex | Evento por ID hexadecimal exato de 64 caracteres. |
| /api/v1/:npub/stats | Total, primeira/última atividade e breakdown por kind. |
| /api/v1/:npub/:kind/hourly | 24 horas de um dia, zero-filled. |
| /api/v1/ids/:hex/related | Eventos com tag e ou q para o ID. |
| /api/v1/:npub/follows | Último kind 3. |
| /api/v1/relay/kinds | Kinds mais frequentes em amostra limitada e visível. |
| /api/v1/relay/top-authors | Autores mais ativos em amostra limitada, com approximate: true. |
| /api/v1/:npub/relays | Último kind 10002. |
| /api/v1/:npub/:kind/monthly | Meses no intervalo since/until, zero-filled e máximo 120. |

### Parâmetros de consulta

| Parâmetro | Regra |
| --- | --- |
| limit | Padrão 100; inteiro positivo; limitado por api.max_limit. |
| offset | Inteiro não negativo; aplicado depois da visibilidade. |
| since, until | Unix timestamp inteiro; since não pode ser maior que until. |
| sort | asc ou ascending significa mais antigo primeiro; padrão é mais novo primeiro. |
| search | NIP-50; alimenta nostr.Filter.Search. |
| e, p, t, d | Filtros por tag; e e p reutilizam normalização NIP-19 do admin. |
| no_p, no_e, no_t, no_d | Excluem eventos que portem a tag, antes de offset e limit. |

Parâmetros malformados retornam 400. IDs hex aceitam exclusivamente 64
caracteres hexadecimais, sem prefixo parcial.

## 4. Arquitetura proposta

    external Fiber app (config.Cfg.Port)
      /api/v1 middleware: enabled, GET, sem WebSocket, concorrência, auth opcional
      /api/v1 handlers: identificador, query, agregações
          -> parser compartilhado: HTTP query/NIP-19 para nostr.Filter
          -> serviço API: consulta paginada e visível
              -> db.QueryEvents / db.CountEvents ou agregações SQL pequenas
              -> groups.QueryEvents(ctx, "", ...)
              -> filtro NIP-70/NIP-59 e exclusões no_*
          -> resposta HTTP / métricas

### Organização de arquivos proposta

- infra/handler/http/api/: handlers, parsing HTTP, respostas e middleware,
  divididos por responsabilidade; nenhum arquivo novo deve ultrapassar cerca de
  300 linhas.
- infra/handler/http/filter/ (ou pacote equivalente): helpers NIP-19 e de tags
  extraídos do admin, sem dependência de Fiber quando possível.
- internal/apiquery/ (ou o pacote api se permanecer pequeno): composição de
  visibilidade, paginação e more. Não criar abstração que não reduza
  complexidade real.
- infra/db/api_*_query.go: agregações SQL específicas e parametrizadas. Não
  mudar QueryEvents para comportamento exclusivo da API.
- infra/metrics/api.go: métricas de baixa cardinalidade por rota e status;
  nunca pubkey, ID, URL ou termo de busca.

O nome final dos pacotes será decidido antes da Fase 1 conforme a proximidade
com os padrões existentes.

## 5. Visibilidade, paginação e contagem

A sequência usada para events, count, more, /count e agregações é a mesma
sequência **visível**. Um evento é invisível se:

1. tiver uma tag NIP-70 [-];
2. for kind 1059, gift wrap NIP-59;
3. for conteúdo de grupo NIP-29 privado, ou metadata de grupo oculta, para
   visitante anônimo, decidido por groups.QueryEvents(ctx, "", ...);
4. possuir uma tag excluída por no_p, no_e, no_t ou no_d.

Hoje o banco limita resultados antes de filtros em Go. Para páginas sem lacunas
ou duplicações, buscar candidatos ordenados de maneira estável (created_at, id)
em blocos; aplicar visibilidade; descartar offset visível; coletar limit + 1
visíveis. Repetir até preencher a página, esgotar a origem ou confirmar more.
A operação deve respeitar cancelamento do contexto e um teto de varredura.

Agregações devem aplicar a mesma visibilidade. Quando a visibilidade não puder
ser expressa integralmente no banco, uma rota só poderá declarar aproximação se
o contrato já a prevê, como relay/top-authors. Contagens que prometem exatidão
não podem ignorar eventos ocultos.

## 6. Configuração proposta para aprovação

    api:
      enabled: false
      max_limit: 0          # 0 = usar security.limits.max_limit
      max_concurrent: 20    # 0 = sem limite dedicado
      nip98:
        enabled: false
        allowed_pubkeys: [] # vazio = qualquer pubkey com NIP-98 válido

- api.max_limit separado evita alterar silenciosamente o limite anunciado em
  NIP-11 para configurar só a API. O fallback reaproveita o limite de
  segurança existente.
- api.enabled igual a false retorna 404 para toda a superfície /api/v1.
- Com NIP-98 desligado, a API continua anônima, conforme nostrfy.
- Com NIP-98 ligado, toda rota /api/v1 exige autorização válida; allowlist
  vazia aceita qualquer assinante e lista não vazia restringe aos pubkeys
  configurados. Validar assinatura, freshness, método GET e URL absoluta;
  nunca registrar o header Authorization.
- A Fase 5.1 deve expor todas as chaves api.* no painel. Persistência e recarga
  serão investigadas antes de codificar, pois Viper costuma exigir reinício.

A configuração resolve a divergência entre API inicialmente anônima e o
requisito posterior de NIP-98: autenticação é opt-in, não mudança de padrão.

## 7. Fases, testes e commits

| Fase | Entrega | Evidência mínima e commit |
| --- | --- | --- |
| 0 | Este documento revisado e aprovado. | Sem código. |
| 1 | Pacote HTTP, GET/WS, identificadores e envelope; extrair parsing compartilhado. | Testes Fiber dos quatro identificadores, curl contra seed, go build ./..., commit feat(api): adiciona rotas públicas por identificador. |
| 2 | query, count, parâmetros, exclusões, ASC, offset e more preparado para visibilidade. | Duas páginas sem sobreposição/gap, limites e inválidos, build. |
| 3 | Agregações por autor/relay, IDs/related e buckets zero-filled. | Testes table-driven de datas, 24h/meses, hex estrito e formatos, build. |
| 4.0 | Ponto único de visibilidade NIP-70/NIP-59/NIP-29 e paginação/contagem visível. | Seeds protegido, gift-wrap e grupo privado ausentes de todas as rotas, build. |
| 4.1 | Habilitação e NIP-98 conforme seção 6. | Casos disabled, assinatura inválida, allowlist e any, build. |
| 5.0 | Erros, limiter, métricas e limite efetivo. | Testes 400/403/404/503, concorrência e métricas, build. |
| 5.1 | Administração de api.* no painel. | Testes do contrato administrativo e da recarga/persistência definida, build. |
| 6 | server.api_host, somente com aprovação separada. | Matriz Host/path, health e metrics, build. |

Ao término de cada fase: listar arquivos, resumir diff, executar go build ./...
e testes focados, criar commit Conventional Commit coeso e parar para aprovação
explícita antes da fase seguinte.

## 8. Decisões pendentes antes de implementar

1. Aprovar o bloco api e seus defaults, principalmente fallback de max_limit e
   concorrência 20.
2. Confirmar que NIP-98, quando ligado, protege toda a API e não rotas
   selecionadas; por padrão continua desligado e anônimo.
3. Aprovar o teto de varredura de paginação visível. Proposta: máximo de
   max(api.max_limit * 20, 2.000) candidatos por request. O comportamento ao
   atingi-lo deve ser definido na Fase 5.0 antes de ser codificado.
4. Aprovar a Fase 6 em pedido separado; ela não inicia automaticamente.

## 9. Não-regressão

- Não registrar /api/v1 na aplicação interna nem tocar Admin/GraphQL ou React,
  exceto a configuração administrativa prevista na Fase 5.1.
- Não criar POST, PUT ou DELETE públicos.
- Não reutilizar limiter do admin; a API terá semáforo próprio.
- Não alterar dto.WsServer.SkipEventFunc nem políticas de escrita para resolver
  visibilidade HTTP. Helpers extraídos devem preservar testes e comportamento
  WebSocket atuais.
- Não incluir payload, busca, token NIP-98, pubkey ou ID como label de métrica
  ou log de rotina.
