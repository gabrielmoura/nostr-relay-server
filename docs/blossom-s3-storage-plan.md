# Plano de armazenamento S3 compatível para Blossom

## Objetivo

Permitir que o armazenamento de blobs Blossom passe do diretório local legado
`files/<sha256>` para um backend compatível com S3, sem mudar contratos HTTP,
metadados PostgreSQL ou semântica de autorização.

## Fase 1 — contrato e configuração

Esta fase introduz `internal/blobstore.Store` e a implementação
`blobstore.LocalStore`. O contrato contém `Put`, `Get` com `ByteRange`, `Stat`,
`Delete` e `List`; `ErrNotFound` e `ErrInvalidRange` são erros portáteis entre
backends. `LocalStore` mantém exatamente o layout existente `files/<sha256>`,
limita gravações ao tamanho declarado mais um byte de detecção, observa
cancelamento durante a cópia e preserva a permissão legada `0644`.

O seletor é `store.backend`, cujo padrão é `local`. A configuração futura do
backend `s3` é declarada em `store.s3`:

- `endpoint`, `region`, `bucket`, `use_path_style` e `key_prefix`;
- `redirect_downloads` e `presign_ttl` (padrão `5m`);
- `fallback_local`;
- `access_key` e `secret_key`.

Com `store.backend: s3`, `endpoint`, `bucket`, `access_key` e `secret_key` são
obrigatórios. `access_key` e `secret_key` devem sempre estar presentes como
par. `NRS_STORE_S3_ACCESS_KEY` e `NRS_STORE_S3_SECRET_KEY`
sobrescrevem os valores do arquivo quando definidos. A saída de `conf print`,
`conf effective` e `conf write` inclui os campos, mas redige as credenciais.

Não há SDK S3, cliente remoto, handler, job ou seleção de backend em runtime
nesta fase. Portanto `store.backend: s3` só valida a configuração; ele ainda
não ativa tráfego S3.

## Fase 2 — backend S3

Adicionar uma implementação `blobstore.S3Store` com SDK compatível, chaves
`<key_prefix>/<sha256>`, operações condicionais/idempotentes, erros mapeados
para `ErrNotFound` e suporte a leitura por intervalo. Testar contra um endpoint
S3 compatível isolado. A integração deve permanecer atrás do contrato criado
na Fase 1.

## Fase 3 — integração do fluxo Blossom

Substituir o acesso direto a `files/<sha256>` nos handlers e nos jobs pelo
`blobstore.Store` selecionado durante o bootstrap. Preservar os checks de
autorização, quota, bloqueio, expiração e os metadados atuais. Adicionar testes
de upload/download, incluindo `Range` e `HEAD`, para ambos os backends.

## Fase 4 — downloads remotos e operação

Quando `redirect_downloads` estiver habilitado, emitir URLs pré-assinadas com
`presign_ttl`; caso contrário, transmitir o conteúdo pelo relay. Definir a
semântica de `fallback_local` somente depois de métricas, timeout, logs
estruturados sem segredos e regras explícitas para evitar divergência de dados.
Documentar migração, rollback e observabilidade antes de habilitar S3 em
produção.
