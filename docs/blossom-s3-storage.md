# Blossom com armazenamento S3 compatível

O backend S3 mantém os contratos Blossom existentes: o relay continua servindo
`GET` e `HEAD /blob/:id`, aplica as mesmas regras de autorização e usa o hash
SHA-256 como identidade do objeto. O bucket precisa existir antes do relay
iniciar; o processo deliberadamente não o cria.

## RustFS com Docker Compose ou Podman Compose

O overlay [`deploy/rustfs/compose.yaml`](../deploy/rustfs/compose.yaml) usa a
imagem oficial `docker.io/rustfs/rustfs:latest` em modo single-node,
single-disk. Ele é opcional e não altera o `docker-compose.yml` padrão.

Defina credenciais novas, longas e distintas — nunca use valores de exemplo ou
credenciais conhecidas — e inicie os serviços:

```bash
export RUSTFS_ACCESS_KEY='troque-por-uma-chave-de-acesso-segura'
export RUSTFS_SECRET_KEY='troque-por-um-segredo-longo-e-unico'
docker compose -f docker-compose.yml -f deploy/rustfs/compose.yaml up -d --build
```

Com Podman Compose, substitua `docker compose` por `podman compose`. A API S3
e o console ficam limitados ao loopback em `127.0.0.1:9000` e
`127.0.0.1:9001`; o relay os alcança pela rede Compose em `rustfs:9000`.

Verifique a API antes de configurar o relay:

```bash
curl --fail http://127.0.0.1:9000/health
```

Crie o bucket uma única vez com um cliente compatível com S3, por exemplo AWS
CLI instalado na máquina administrativa:

```bash
aws --endpoint-url http://127.0.0.1:9000 s3api create-bucket --bucket nostr-blobs
```

O volume nomeado evita o problema de permissões de bind mounts. Para um bind
mount, o diretório precisa estar gravável pelo UID/GID `10001`, usado pelo
container RustFS. Consulte a [documentação oficial de containers do
RustFS](https://docs.rustfs.com/en/installation/container) antes de expor a API
S3 ou o console fora da máquina.

## Configuração do relay

Mantenha segredos fora de `conf.yaml` e injete-os no processo do relay:

```bash
export NRS_STORE_S3_ACCESS_KEY="$RUSTFS_ACCESS_KEY"
export NRS_STORE_S3_SECRET_KEY="$RUSTFS_SECRET_KEY"
```

Para Compose, acrescente as duas variáveis ao serviço `nrserver` por meio de um
arquivo de ambiente do seu deploy. Não as inclua no repositório nem em logs.

Configure o restante em `conf.yaml` (no Compose, o endpoint usa o nome do
serviço):

```yaml
store:
  enabled: true
  backend: s3
  media_path: http://localhost:9090/blob
  s3:
    endpoint: http://rustfs:9000
    bucket: nostr-blobs
    use_path_style: true
    key_prefix: blossom
    fallback_local: false
```

`fallback_local` só atende uma leitura quando S3 responde explicitamente que o
objeto não existe. Ele não transforma uma falha do S3 em leitura local e não
duplica escritas; isso evita mascarar indisponibilidade ou criar conjuntos de
dados divergentes.

## Migração e ativação

1. Faça backup de `files/` e mantenha o relay em modo local durante a cópia.
2. Execute primeiro uma simulação:

   ```bash
   NRS_STORE_S3_ACCESS_KEY="$RUSTFS_ACCESS_KEY" \
   NRS_STORE_S3_SECRET_KEY="$RUSTFS_SECRET_KEY" \
   nrserver blossom migrate --dry-run --workers 4
   ```

3. Execute a cópia. Ela não remove arquivos locais e falha se encontrar um
   objeto remoto com o mesmo hash e tamanho diferente:

   ```bash
   NRS_STORE_S3_ACCESS_KEY="$RUSTFS_ACCESS_KEY" \
   NRS_STORE_S3_SECRET_KEY="$RUSTFS_SECRET_KEY" \
   nrserver blossom migrate --workers 4
   ```

4. Altere `store.backend` para `s3` e reinicie o relay. Habilite
   `fallback_local: true` apenas durante a janela de transição, caso objetos
   antigos ainda não tenham sido copiados.
5. Depois de verificar a cópia e os downloads, desabilite o fallback. Só então
   planeje a retenção ou remoção do diretório local a partir de um backup
   verificado.

Para rollback, pare o relay, restaure `store.backend: local`, mantenha o
diretório `files/` intacto e reinicie. Objetos aceitos somente depois da troca
para S3 não estarão no armazenamento local; recopie-os antes de depender do
rollback para atendimento completo.

## Verificação HTTP e Range

Use um hash que já exista no banco de metadados e no bucket. Os comandos abaixo
confirmam que a interface pública não mudou:

```bash
curl -i "http://localhost:9090/blob/<sha256>"
curl -I "http://localhost:9090/blob/<sha256>"
curl -i -H 'Range: bytes=0-99' "http://localhost:9090/blob/<sha256>"
curl -i -H 'Range: bytes=999999999-' "http://localhost:9090/blob/<sha256>"
```

Os dois primeiros retornam `200`; o pedido parcial válido retorna `206` com
`Accept-Ranges: bytes` e `Content-Range`; o último retorna `416` quando o
intervalo não é satisfatível. Compare o hash do corpo baixado integralmente com
o identificador do blob.

## Métricas

O endpoint Prometheus expõe:

- `nostr_blob_store_operations_total{backend,operation,result}`
- `nostr_blob_store_operation_duration_seconds{backend,operation,result}`

`backend` é `local` ou `s3`; `operation` é `put`, `get`, `stat`, `delete` ou
`list`; `result` é `success`, `not_found`, `invalid_range`, `canceled` ou
`error`. Nenhum rótulo contém hash, bucket, endpoint ou mensagem de erro.

Exemplo de consulta para erros S3 nos últimos cinco minutos:

```promql
sum(rate(nostr_blob_store_operations_total{backend="s3",result="error"}[5m])) by (operation)
```
