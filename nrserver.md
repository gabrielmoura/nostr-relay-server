# Nostr Relay Server

Documentacao unificada para preparar, configurar, migrar e executar o `nrserver`.

## Visao Geral

O `nrserver` e um relay Nostr em Go com:

- armazenamento principal em PostgreSQL (obrigatorio)
- Redis opcional (cache/pubsub)
- painel/admin interno em `/panel` e `/admin`
- NIP-86 opcional na raiz publica com autenticacao NIP-98
- suporte a import/export, download e sync por CLI
- metricas Prometheus em `/metrics`

## Pre-requisitos

- Go `1.24+`
- PostgreSQL em execucao e acessivel
- Redis (opcional)

Opcional para ambiente com servico:

- `systemd`
- Docker / Docker Compose (se usar topologia containerizada)

## Preparacao Rapida (Local)

### 1) Clonar e entrar no projeto

```bash
git clone https://github.com/gabrielmoura/nostr-relay-server.git
cd nostr-relay-server
```

### 2) Gerar arquivo de configuracao

```bash
go run ./cmd/nrserver conf write
```

Esse comando cria `conf.yaml`.

### 3) Ajustar `conf.yaml`

No minimo, revise:

- `db.postgres_uri` (obrigatorio)
- `port` (porta externa do relay)
- `relay_information.*` (dados NIP-11)
- `admin_token` (recomendado para proteger `/admin/*`)
- `nip86.enabled` e `admin_pubkey` somente se voce realmente precisar de gerenciamento remoto Nostr-native

Exemplo de URI:

```yaml
db:
  postgres_uri: postgres://postgres:Strong@P4ssword@127.0.0.1:5432/nostr
```

### 4) Validar configuracao

```bash
go run ./cmd/nrserver conf validate
```

### 5) Rodar migration (seed)

```bash
go run ./cmd/nrserver seed
```

Esse passo prepara o schema do banco.

### 6) Subir o relay

```bash
go run ./cmd/nrserver server
```

Opcional: iniciar com bootstrap de eventos iniciais:

```bash
go run ./cmd/nrserver server --bootstrap
```

## Endpoints Padrao

Com `port: 9090` no `conf.yaml`:

- Relay/NIP-11: `http://localhost:9090`
- NIP-86 JSON-RPC (opcional): `POST http://localhost:9090/`
- Admin API: `http://localhost:9091/admin`
- Admin Panel: `http://localhost:9091/panel`
- Metrics: `http://localhost:9091/metrics`

Se `admin_token` estiver definido, envie header:

```text
X-Admin-Token: <seu_token>
```

Para NIP-86:

- manter desabilitado por padrao e recomendado para a maioria dos usuarios
- quando habilitado, exige `admin_pubkey` e `relay_information.url` corretos
- usa `Content-Type: application/nostr+json+rpc` e `Authorization: Nostr <base64-event>`

## Fluxo com Binario

### Build

```bash
go build -o nrserver ./cmd/nrserver
```

### Comandos equivalentes

```bash
./nrserver conf write
./nrserver conf validate
./nrserver seed
./nrserver server
```

## Passo a Passo para Ambiente de Servico (Linux + systemd)

### 1) Criar diretorios

```bash
sudo mkdir -p /etc/nrs /opt/nrs/nrserver
```

### 2) Instalar binario

```bash
sudo cp ./nrserver /opt/nrs/nrserver/nrserver
```

### 3) Gerar config em `/etc/nrs/conf.yaml`

```bash
sudo /opt/nrs/nrserver/nrserver conf write --file /etc/nrs/conf.yaml --force
sudo /opt/nrs/nrserver/nrserver conf validate --file /etc/nrs/conf.yaml
```

### 4) Rodar migration apontando para config de sistema

```bash
sudo /opt/nrs/nrserver/nrserver seed
```

> O loader procura `conf.yaml` em `.`, `../..` e `/etc/nrs`.

### 5) Instalar unit file e iniciar servico

```bash
sudo cp ./nrserver.service /etc/systemd/system/nrserver.service
sudo systemctl daemon-reload
sudo systemctl enable nrserver
sudo systemctl start nrserver
sudo systemctl status nrserver
```

## Comandos Iniciais Essenciais

### Configuracao (`conf`)

```bash
nrserver conf print
nrserver conf effective
nrserver conf validate
nrserver conf write --file ./conf.yaml --force
```

### Migration e bootstrap (`seed`)

```bash
nrserver seed
nrserver seed --bootstrap
nrserver seed --bootstrap --bootstrap-idempotent
nrserver seed --dry-run
```

### Execucao (`server`)

```bash
nrserver server
nrserver server --bootstrap
```

### Manutencao (`cron`)

```bash
nrserver cron --list
nrserver cron --run-once
nrserver cron --run-once --job db_optimization
```

### Mobilidade de dados

Importar:

```bash
nrserver import --file events.jsonl
```

Exportar:

```bash
nrserver export --file export.jsonl
nrserver export --format tsv --file events.tsv
```

Download de relays:

```bash
nrserver download --relay-url wss://relay.damus.io --public-key <hex-ou-npub>
```

Sync Negentropy:

```bash
nrserver sync wss://relay.damus.io --pk <hex-ou-npub> --dir both
```

## Verificacao Pos-Setup

### 1) Validar health basico

```bash
curl -sS http://localhost:9090
curl -sS http://localhost:9091/metrics
```

### 2) Conferir logs em desenvolvimento

```bash
go run ./cmd/nrserver server
```

### 3) Conferir logs em systemd

```bash
journalctl -u nrserver -f
```

## Ambiente Docker + Tor (Opcional)

Para topologia com `docker-compose`, PostgreSQL e Onion Service, use como base o guia:

- `Docker_Tor.md`

Fluxo resumido:

1. criar `docker-compose.yml`, `conf.yaml`, `torrc` e segredo do Postgres
2. subir com `docker-compose up -d`
3. obter hostname `.onion`
4. atualizar `conf.yaml` (`url`, `canonical_url`, `icon`, `api_path`, `media_path`)
5. reiniciar `nostr-relay`

## Solucao Rapida de Problemas

- erro `missing DB URI`: preencher `db.postgres_uri` no `conf.yaml`
- erro de cron expression: usar formato com 6 campos (`sec min hour day month weekday`)
- erro de `/admin` sem token: enviar `X-Admin-Token` quando `admin_token` estiver configurado
- erro de NIP-86: validar `admin_pubkey`, `relay_information.url` e o hash `payload` do body
- `seed` com falha de conexao: validar acesso ao PostgreSQL (host, porta, usuario, senha, DB)

## Limitacoes importantes

- PostgreSQL e obrigatorio.
- Redis continua opcional.
- `blockip` do NIP-86 derruba imediatamente apenas conexoes conhecidas pelo processo local.
- overrides de metadata do relay persistem no banco, nao reescrevem `conf.yaml`.
- se voce so precisa do painel embedado, nao ha motivo tecnico para habilitar NIP-86.

## Referencias Consolidada

- `README.md`
- `install.md`
- `Docker_Tor.md`
- `docs/configuration.md`
- `docs/cli.md`
- `docs/api-spec.md`
- `docs/data-schema.md`
- `docs/architecture.md`
- `docs/download-command.md`
- `docs/policies.md`
- `docs/decisions.md`
