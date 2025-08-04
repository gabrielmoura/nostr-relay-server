## Configurando um Ambiente Nostr Relay com Docker Compose, PostgreSQL e Tor

Este guia detalha como implantar um ambiente completo para um Nostr Relay utilizando Docker Compose. A configuração
inclui o próprio Nostr Relay, um banco de dados PostgreSQL para persistência de dados e um serviço Tor para expor o
relay como um Onion Service, garantindo maior privacidade e anonimato.

### Estrutura do Projeto

Para começar, crie a seguinte estrutura de diretórios e arquivos no seu projeto:

```
.
├── docker-compose.yml
├── conf.yaml
├── tor-data/
│   └── torrc/
│       └── torrc
└── Dockerfile  # Seu Dockerfile para a imagem nostr-relay
```

### 1. `docker-compose.yml`

O arquivo `docker-compose.yml` é o coração do nosso ambiente. Ele orquestra a criação e a interconexão dos
contêineres: `nostr-relay`, `postgres` e `tor`.

**Melhorias e Explicações:**

* **Segredos do Docker:** Para gerenciar a senha do PostgreSQL de forma mais segura, utilizamos `secrets` do Docker em
  vez de variáveis de ambiente em texto plano.
* **Redes Detalhadas:** As redes `net-int` (para comunicação interna entre o relay e o banco de dados) e `tor-test` (
  para a comunicação entre o Tor e o relay) estão claramente definidas com sub-redes estáticas para evitar conflitos.
* **Segurança:** O serviço `nostr-relay` é configurado com `no-new-privileges:true` e `cap_drop: - ALL` para reduzir a
  superfície de ataque do contêiner. O usuário dentro do contêiner também é definido como não-root (`1000:1000`).
* **Volumes Persistentes:** Volumes são usados para persistir os dados do PostgreSQL (`./postgres_data`), a configuração
  do Tor (`./tor-data`) e o arquivo de configuração do relay (`./conf.yaml`).

```yml
version: '3.8'

# Define o segredo para a senha do banco de dados
secrets:
  postgres_password:
    file: ./postgres_password.txt

services:
  # Serviço do Nostr Relay
  nostr-relay:
    container_name: nostr-relay
    image: nostr-relay:latest # Garanta que esta imagem foi construída ou está disponível
    build:
      context: .
      dockerfile: Dockerfile
    restart: always
    user: "1000:1000" # Executa como um usuário não-root
    ports:
      # Porta do Relay exposta para o Tor
      - "9090:9090"
      # Interface de administração acessível apenas localmente
      - "127.0.0.1:9091:9091"
    volumes:
      # Monta o arquivo de configuração como somente leitura
      - ./conf.yaml:/app/conf.yaml:ro
    environment:
      # Variável de ambiente para o proxy Tor
      HTTP_PROXY: "socks5://10.6.0.4:9050"
      DB_HOST: "postgres" # Utiliza o nome do serviço para resolução de DNS
    networks:
      net-int:
        ipv4_address: 10.5.0.9
      tor-test:
        ipv4_address: 10.6.0.5
    depends_on:
      - postgres
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL

  # Serviço do Banco de Dados PostgreSQL
  postgres:
    container_name: postgres
    image: postgres:16.1-alpine
    restart: always
    environment:
      # A senha é lida do arquivo de segredos
      POSTGRES_PASSWORD_FILE: /run/secrets/postgres_password
      LANG: pt_BR.UTF-8
      TZ: America/Sao_Paulo
      POSTGRES_INITDB_ARGS: "--locale-provider=icu --icu-locale=pt-BR"
    volumes:
      # Volume para persistir os dados do PostgreSQL
      - ./postgres_data:/var/lib/postgresql/data
    networks:
      net-int:
        ipv4_address: 10.5.0.2
    ports:
      # Porta do PostgreSQL exposta apenas localmente para administração
      - "127.0.0.1:5432:5432"
    secrets:
      - postgres_password

  # Serviço do Tor
  tor:
    container_name: tor
    image: osminogin/tor-simple:latest
    restart: always
    volumes:
      # Volume para os dados do serviço Tor (incluindo o hostname)
      - ./tor-data:/var/lib/tor
      # Monta o arquivo de configuração do Tor como somente leitura
      - ./torrc:/etc/tor/torrc:ro
    networks:
      tor-test:
        ipv4_address: 10.6.0.4

# Definição das Redes
networks:
  net-int:
    driver: bridge
    ipam:
      config:
        - subnet: 10.5.0.0/24
          gateway: 10.5.0.1
  tor-test:
    driver: bridge
    ipam:
      config:
        - subnet: 10.6.0.0/24
          gateway: 10.6.0.1
```

### 2. Arquivo de Senha do PostgreSQL

Para usar os `secrets` do Docker, crie um arquivo chamado `postgres_password.txt` no mesmo diretório
do `docker-compose.yml` e insira sua senha desejada.

**`postgres_password.txt`**:

```
Strong@P4ssword
```

**Importante:** Adicione `postgres_password.txt` ao seu arquivo `.gitignore` para evitar que a senha seja enviada para o
seu repositório Git.

### 3. Configuração do Nostr Relay (`conf.yaml`)

Este arquivo configura o comportamento do seu relay. Substitua os valores entre colchetes `[seu_endereco]` pelo
endereço `.onion` que será gerado pelo Tor. Você precisará iniciar os serviços uma vez para obter este endereço e depois
atualizar o arquivo.

```yaml
port: 9090
app_env: production
ws:
  rate_limit: 1
  burst: 5
  auth: false

# Informações do Relay (NIP-11)
relay_information:
  # ATUALIZE APÓS OBTER O ENDEREÇO ONION
  url: "http://[seu_endereco].onion"
  name: "Nostr Relay Server"
  description: "A Nostr Relay Server operando na rede Tor."
  pub_key: "7ef721e77149c737014971b141b0b590a5ebe82b79130228cdbe56e9be2d8e50"
  priv_key: "e4d347b85fe3429ac1995d3eab801a3858990f66fb0"
  supported_nips: [ 1, 2, 4, 9, 11, 17, 25, 45 ]
  software: "https://github.com/gabrielmoura/nostr-relay-server"
  version: "0.1.0"
  # ATUALIZE APÓS OBTER O ENDEREÇO ONION
  canonical_url: "ws://[seu_endereco].onion"
  # ATUALIZE APÓS OBTER O ENDEREÇO ONION
  icon: "http://[seu_endereco].onion/nostr.png"

# Configurações do Relay
relay:
  query_limit: 100
  query_ids_limit: 500
  query_authors_limit: 500
  query_kinds_limit: 10
  query_tags_limit: 100
  keep_recent_events: true
  max_size_event_in_bytes: 100000
  filter_limit: 9999999999
  reporting_limit: 5
  enable_anonymous_req: true
  max_tag_value_length: 150

# Configurações do Banco de Dados
db:
  max_conns: 10
  min_conns: 1
  # Utiliza o nome do serviço 'postgres' e lê a senha do arquivo de segredos
  postgres_uri: "postgres://postgres:Strong@P4ssword@postgres:5432/nostr"

# Configurações de Streaming (opcional)
stream:
  relays:
    - "ws://127.0.0.1:7777"
    - "ws://node-ior.webhop.me:6000"
  stream_down: true
  stream_up: true

# Configurações de Armazenamento (NIP-96)
store:
  enabled: true
  # ATUALIZE APÓS OBTER O ENDEREÇO ONION
  api_path: "http://[seu_endereco].onion/upload"
  media_path: "http://[seu_endereco].onion/blob"
  accepted_mimetypes:
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "image/webp"
    - "image/svg+xml"
    - "video/mp4"
    - "video/webm"
    - "audio/mpeg"
    - "audio/opus"
  allow_adult_content: false
  allow_violent_content: false
  names: [ ]
```

### 4. Configuração do Tor (`torrc`)

Este arquivo instrui o Tor a criar um serviço oculto (Hidden Service) e a redirecionar o tráfego da porta 80 para a
porta 9090 do nosso contêiner `nostr-relay`.

Crie o arquivo em `tor-data/torrc/torrc`:

```ini
# Expõe um SOCKS proxy na rede interna do Tor para uso do nostr-relay
SOCKSPort 0.0.0.0:9050

# Diretório onde o Tor armazenará as chaves e o hostname do serviço oculto
HiddenServiceDir /var/lib/tor/nostr-relay

# Mapeia a porta 80 do serviço .onion para o endereço IP e porta do nostr-relay
HiddenServicePort 80 10.6.0.5:9090
```

### 5. Como Executar

#### Passo 1: Construir a Imagem (se necessário)

Se você estiver usando um `Dockerfile` local para construir sua imagem `nostr-relay`, execute:

```bash
docker-compose build
```

#### Passo 2: Iniciar os Serviços

Para iniciar todos os serviços em segundo plano, utilize o comando:

```bash
docker-compose up -d
```

#### Passo 3: Obter o Endereço Onion

O Tor gerará um endereço `.onion` único para o seu serviço. Para descobri-lo, execute o seguinte comando após os
contêineres estarem em execução:

```bash
docker-compose exec tor cat /var/lib/tor/nostr-relay/hostname
```

O resultado será algo como `abcdef1234567890.onion`.

#### Passo 4: Atualizar `conf.yaml` e Reiniciar

Copie o endereço `.onion` obtido no passo anterior e cole-o nos campos `url`, `canonical_url`, `icon`, `api_path`
e `media_path` do seu arquivo `conf.yaml`.

Após salvar as alterações, reinicie o serviço `nostr-relay` para que ele carregue a nova configuração:

```bash
docker-compose restart nostr-relay
```

### Gerenciamento do Ambiente

* **Verificar logs:**
  ```bash
  # Ver logs de todos os serviços
  docker-compose logs -f

  # Ver logs de um serviço específico (ex: nostr-relay)
  docker-compose logs -f nostr-relay
  ```
* **Parar os serviços:**
  ```bash
  docker-compose down
  ```
* **Parar e remover volumes (cuidado, isso apagará todos os dados do PostgreSQL):**
  ```bash
  docker-compose down -v
  ```