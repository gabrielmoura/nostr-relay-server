# NRServer + Tor com Docker Compose e Swarm

Este guia mostra como executar o `nrserver` com PostgreSQL, Redis e Tor (onion service) usando o `Dockerfile` multi-stage do projeto, o `docker-compose.yml` da raiz e o `stack.yml` para Docker Swarm.

## 1) O que foi padronizado

- Build da imagem do relay via `Dockerfile` (backend Go + frontend Node).
- Orquestracao com `docker-compose.yml`:
  - `nrserver` (relay)
  - `postgres` (persistencia)
  - `redis` (cache, pub/sub e jobs)
  - `tor` (hidden service e socks proxy)
- Orquestracao opcional com `stack.yml` para Docker Swarm.
- Configuracao do Tor em `torrc`.
- Senha do PostgreSQL via `secret` local (`deploy/postgres/password.txt`).
- Arquivos `conf.yaml`, `postgresql.conf`, `redis.conf` e `torrc` entregues via `configs` do Docker.

## 2) Arquivos usados

```
.
├── Dockerfile
├── docker-compose.yml
├── stack.yml
├── conf.yaml
├── deploy/
│   ├── postgres/
│   │   ├── postgresql.conf
│   │   ├── password.txt          # criar localmente (nao comitar)
│   │   └── password.txt.example
│   └── redis/
│       └── redis.conf
├── torrc
└── Docker_Tor.md
```

## 3) Pre-requisitos

- Docker 24+
- Docker Compose v2 (`docker compose`)
- BuildKit/Buildx habilitado (padrao nas versoes recentes)

## 4) Preparar configuracao

### 4.1 Ajuste o arquivo de senha do Postgres

O repositório já inclui `deploy/postgres/password.txt` com um valor placeholder.

Troque esse valor antes de subir o ambiente:

```bash
printf 'uma-senha-forte\n' > deploy/postgres/password.txt
```

### 4.2 Ajuste o `conf.yaml`

No `conf.yaml`, configure banco e Redis para usar os servicos da rede compose/swarm:

```yaml
db:
  postgres_uri: postgres://postgres:SUA_SENHA@postgres:5432/nostr?sslmode=disable

redis:
  enabled: true
  addr: redis:6379
  password: ""
```

Revise tambem os arquivos usados como `configs`:

- `deploy/postgres/postgresql.conf`
- `deploy/redis/redis.conf`
- `torrc`

Se quiser trafego de saida via Tor, mantenha os proxies `HTTP_PROXY`, `HTTPS_PROXY` e `ALL_PROXY` definidos no servico `nrserver` do compose/stack.

## 5) Build e subida do ambiente

### 5.1 Subir tudo com build

```bash
docker compose up -d --build
```

### 5.2 Logs

```bash
docker compose logs -f nrserver
docker compose logs -f redis
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

## 7) Deploy com Docker Swarm

### 7.1 Inicialize o Swarm

```bash
docker swarm init
```

### 7.2 Garanta os mesmos arquivos locais

Arquivos necessarios:

- `./conf.yaml`
- `./deploy/postgres/postgresql.conf`
- `./deploy/postgres/password.txt`
- `./deploy/redis/redis.conf`
- `./torrc`

### 7.3 Suba a stack

```bash
docker stack deploy -c stack.yml nrserver
```

### 7.4 Acompanhe a stack

```bash
docker stack services nrserver
docker service logs -f nrserver_nrserver
docker service logs -f nrserver_tor
```

## 8) Buildx com cache (CI/local)

Para build manual da imagem com cache remoto:

```bash
docker buildx build \
  --platform linux/amd64 \
  --cache-from type=registry,ref=ghcr.io/seu-user/nostr-relay-server:buildcache \
  --cache-to type=registry,ref=ghcr.io/seu-user/nostr-relay-server:buildcache,mode=max \
  -t ghcr.io/seu-user/nostr-relay-server:latest \
  .
```

## 9) Operacao diaria

Parar mantendo dados:

```bash
docker compose down
```

Parar removendo dados persistidos (cuidado):

```bash
docker compose down -v
```

Remover stack no Swarm:

```bash
docker stack rm nrserver
```

## 10) Observacoes de seguranca

- Troque imediatamente o valor placeholder de `deploy/postgres/password.txt`.
- Em producao, mantenha `admin_token` definido.
- Exponha `9091` apenas localmente (ja configurado como `127.0.0.1:9091:9091`).
- Revise os campos `relay_information` antes de publicar o relay.
- Se nao precisar trafego de saida via Tor, remova os proxies do servico `nrserver`.
