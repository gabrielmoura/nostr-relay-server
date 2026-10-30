# Cordn e Concord — TODO de implementação

Este checklist descreve o caminho aprovado para manter os dois protocolos no
binário único do NRServer. Não habilita funcionalidades por si só.

## Fase 0 — contratos e limites

- [ ] Congelar a versão das specs Cordn e Concord usada na compatibilidade.
- [x] Definir `cordn.enabled` e `concord.enabled`, ambas com default `false`.
- [ ] Definir limites de payload, backlog, streams, retenção e rate limit.
- [ ] Definir códigos de erro externos e mensagens seguras.
- [ ] Registrar explicitamente a fronteira: Cordn/Concord não usam RelaySync.
- [ ] Definir a política de acesso das superfícies administrativas e exportações.

## Fase 1 — coordinator Cordn no processo Go

- [x] Criar `internal/cordn` com contratos de transporte, serviço e storage.
- [x] Implementar transporte ContextVM/Nostr plaintext para pedidos dirigidos ao relay.
- [x] Implementar autenticação da requisição pela assinatura Nostr validada antes do handler.
- [x] Validar assinatura da publicação e vínculo BasicCredential/signer.
- [x] Implementar métodos de KeyPackage: publish, list, consume e remove.
- [x] Implementar filas de Welcome e Join Requests com consumo por referência temporal.
- [x] Implementar mensagens MLS opacas, cursor monotônico por `gid` e backlog durável.
- [x] Registrar assinantes antes do replay para eliminar a janela de perda.
- [ ] Implementar CEP-41/Open Stream interoperável para entrega live e cancelamento remoto.
- [ ] Adicionar rate limit por identidade e quotas por identidade/grupo.

## Fase 2 — persistência Cordn

- [x] Criar migration para KeyPackages e referências de publicação.
- [x] Criar migration para Welcomes e Join Requests pendentes/consumidos.
- [x] Criar migration para mensagens por `gid` e cursor.
- [x] Garantir unicidade e avanço transacional do cursor por grupo.
- [x] Implementar retenção e limpeza de filas/mensagens sem reutilizar tabelas RelaySync.
- [ ] Adicionar interfaces de storage injetáveis para o serviço.
- [ ] Integrar pool PostgreSQL e shutdown do coordinator no `server`.

## Fase 3 — Concord como política de relay

- [x] Definir o filtro seguro de eventos Concord sem decriptar giftwraps.
- [x] Aplicar a política ao backlog de `REQ`, `COUNT` e notificações live originadas da assinatura validada.
- [ ] Revisar chaves de cache para incluir o escopo de visibilidade.
- [ ] Revisar exportação, busca, GraphQL e superfícies administrativas.
- [ ] Adicionar quotas, limites de filtros e rate limit para `kind:1059`.
- [ ] Manter ingestão Nostr genérica e não interpretar membership/roles.

## Fase 4 — bootstrap e configuração

- [x] Adicionar blocos `cordn` e `concord` à configuração tipada.
- [x] Validar pré-requisitos somente quando a flag correspondente estiver ativa.
- [x] Inicializar o coordinator no ciclo de vida do `cmd/server.go`.
- [ ] Emitir logs estruturados com protocolo, fase e identidade truncada.
- [ ] Garantir que tokens, chaves e payloads privados não apareçam nos logs.
- [x] Atualizar exemplo de configuração e documentação operacional.

## Fase 5 — compatibilidade e segurança

- [ ] Validar publicação/consumo de KeyPackages com cliente Cordn de referência.
- [ ] Validar Welcome, Join Request, fetch e subscription live.
- [ ] Validar retomada após reinício em cada estado do coordinator.
- [ ] Validar deduplicação na fronteira replay/live.
- [ ] Validar cancelamento sem goroutines ou assinantes órfãos.
- [ ] Validar isolamento de dados entre `gid`s e identidades.
- [ ] Validar que Concord não expõe giftwraps por REQ, COUNT ou live indevidos.
- [ ] Executar `go build ./...` e `go vet ./...`.
- [ ] Executar validação de compatibilidade externa antes de alterar defaults.

## Fora do escopo

- [ ] Não alterar o protocolo, storage ou lifecycle do RelaySync.
- [ ] Não usar RelaySync como transporte Cordn ou Concord.
- [ ] Não implementar decriptação MLS/Concord no relay.
- [ ] Não transformar Concord em autoridade server-side de comunidades.
