# NRServer + Tor com Docker Compose

Este guia mostra como executar o `nrserver` com PostgreSQL e Tor (onion service) usando o `Dockerfile` multi-stage do projeto e o `docker-compose.yml` da raiz.

## 1) O que foi padronizado

- Build da imagem do relay via `Dockerfile` (backend Go + frontend Node).
- Orquestracao com `docker-compose.yml`:
  - `nrserver` (relay)
  - `postgres` (persistencia)
  - `tor` (hidden service e socks proxy)
- Configuracao do Tor em `torrc`.
- Senha do PostgreSQL via secret local (`postgres_password.txt`).

## 2) Arquivos usados

```
.
├── Dockerfile
├── docker-compose.yml
├── conf.yaml
├── torrc
└── postgres_password.txt   # criar localmente (nao comitar)
```

## 3) Pre-requisitos

- Docker 24+
- Docker Compose v2 (`docker compose`)
- BuildKit/Buildx habilitado (padrao nas versoes recentes)

## 4) Preparar configuracao

### 4.1 Crie o arquivo de senha do Postgres

Use o exemplo versionado e gere seu arquivo local:

```bash
cp postgres_password.txt.example postgres_password.txt
```

Depois altere o valor para uma senha forte.

### 4.2 Ajuste o `conf.yaml`

No `conf.yaml`, configure o banco para usar o servico `postgres` da rede compose:

```yaml
db:
  postgres_uri: postgres://postgres:SUA_SENHA@postgres:5432/nostr
```

Se quiser trafego de saida via Tor, mantenha os proxies definidos no servico `nrserver` do compose.

## 5) Build e subida do ambiente

### 5.1 Subir tudo com build

```bash
docker compose up -d --build
```

### 5.2 Logs

```bash
docker compose logs -f nrserver
docker compose logs -f tor
docker compose logs -f postgres
```

## 6) Obter endereco onion

Depois que o container `tor` estiver iniciado:

```bash
docker compose exec tor cat /var/lib/tor/nostr-relay/hostname
```

Exemplo de retorno:

```
abcdefgh12345678abcdefgh12345678abcdefgh12345678abcdefgh.onion
```

Atualize no `conf.yaml` os campos de URL publica (NIP-11 e store), por exemplo:

- `relay_information.url`
- `relay_information.canonical_url`
- `relay_information.icon`
- `store.api_path`
- `store.media_path`

Reinicie o relay:

```bash
docker compose restart nrserver
```

## 7) Buildx com cache (CI/local)

Para build manual da imagem com cache remoto:

```bash
docker buildx build \
  --platform linux/amd64 \
  --cache-from type=registry,ref=ghcr.io/seu-user/nostr-relay-server:buildcache \
  --cache-to type=registry,ref=ghcr.io/seu-user/nostr-relay-server:buildcache,mode=max \
  -t ghcr.io/seu-user/nostr-relay-server:latest \
  .
```

## 8) Operacao diaria

Parar mantendo dados:

```bash
docker compose down
```

Parar removendo dados persistidos (cuidado):

```bash
docker compose down -v
```

## 9) Observacoes de seguranca

- Nao comite `postgres_password.txt`.
- Em producao, mantenha `admin_token` definido.
- Exponha `9091` apenas localmente (ja configurado como `127.0.0.1:9091:9091`).
- Revise os campos `relay_information` antes de publicar o relay.
