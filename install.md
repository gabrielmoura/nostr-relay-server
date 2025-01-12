# Instalação

## Manual

Para instalar o projeto manualmente, siga os passos abaixo:

Crie os diretórios necessários:

```bash
mkdir -p /etc/nrs /opt/nrs/nrserver
```

Crie o unit file do systemd:

```bash
cp  nrserver.service /etc/systemd/system/nrserver.service
```

Imprima o arquivo de configuração padrão edite e salve em `/etc/nrs/conf.yaml`:

```bash
./nrserver conf print
```
