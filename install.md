# Instalação

## Manual

Para instalar o projeto manualmente, siga os passos abaixo.

### 1. Criar diretórios

```bash
mkdir -p /etc/nrs /opt/nrs/nrserver
```

### 2. Instalar o binário

```bash
go build -o nrserver ./cmd/nrserver
install -m 0755 nrserver /opt/nrs/nrserver/nrserver
```

### 3. Criar a configuração

Imprima a configuração padrão, edite e salve em `/etc/nrs/conf.yaml`:

```bash
/opt/nrs/nrserver/nrserver conf print > /etc/nrs/conf.yaml
```

No mínimo, revise:

* `db.postgres_uri`
* `port`
* `relay_information.url`
* `relay_information.canonical_url`
* `admin_token`
* `redis.enabled`, `redis.addr` e `redis.password` se Redis estiver habilitado

### 4. Instalar o unit file do systemd

```bash
cp nrserver.service /etc/systemd/system/nrserver.service
systemctl daemon-reload
systemctl enable nrserver
```

### 5. Preparar o banco

```bash
/opt/nrs/nrserver/nrserver seed
```

### 6. Subir o serviço

```bash
systemctl start nrserver
systemctl status nrserver
```

## Docker Compose

O projeto fornece `docker-compose.yml` com:

* `nrserver`
* `postgres:18.4-alpine3.23`
* `redis:8.8.0-alpine3.23`
* `tor`

Os manifests usam `configs` e `secrets` do Docker para entregar arquivos de configuracao aos containers.

### 1. Preparar os arquivos de configuração

O repositório já inclui `deploy/postgres/password.txt` com um valor placeholder.

Troque esse valor antes de subir qualquer ambiente compartilhado:

```bash
printf 'uma-senha-forte\n' > deploy/postgres/password.txt
```

Revise os arquivos:

* `conf.yaml`
* `deploy/postgres/postgresql.conf`
* `deploy/redis/redis.conf`
* `torrc`

### 2. Ajustar `conf.yaml`

Exemplo minimo para Compose:

```yaml
db:
  postgres_uri: postgres://postgres:SUA_SENHA@postgres:5432/nostr?sslmode=disable

redis:
  enabled: true
  addr: redis:6379
  password: ""
```

### 3. Build e subida

```bash
docker compose up -d --build
```

### 4. Verificações úteis

```bash
docker compose logs -f nrserver
curl -i -H "Accept: application/nostr+json" http://localhost:9090/
curl -i http://localhost:9091/metrics
curl -i http://localhost:9091/panel
```

### 5. Obter o hostname onion

```bash
docker compose exec tor cat /var/lib/tor/nostr-relay/hostname
```

Depois atualize no `conf.yaml` os campos públicos relevantes:

* `relay_information.url`
* `relay_information.canonical_url`
* `relay_information.icon`
* `store.api_path`
* `store.media_path`

E reinicie o relay:

```bash
docker compose restart nrserver
```

## Docker Swarm

O projeto também fornece `stack.yml` para deploy em Swarm.

### 1. Inicializar o cluster

```bash
docker swarm init
```

### 2. Preparar os mesmos arquivos locais

Garanta a presença de:

* `./conf.yaml`
* `./deploy/postgres/postgresql.conf`
* `./deploy/postgres/password.txt`
* `./deploy/redis/redis.conf`
* `./torrc`

### 3. Subir a stack

```bash
docker stack deploy -c stack.yml nrserver
```

### 4. Acompanhar o deploy

```bash
docker stack services nrserver
docker service logs -f nrserver_nrserver
```
