# Nostr Relay Server


### Rodando Prometheus
```yml
global:
scrape_interval:     15s
scrape_configs:
- job_name: 'nostr'
  scrape_interval: 5s
  static_configs:
  - targets: ['host.docker.internal:9090']
```
```bash
docker run -d \
-v./prom.yml:/etc/prometheus/prometheus-scrape-config.yaml \
-p 8080:9090 --add-host=host.docker.internal:host-gateway  prom/prometheus \
 --config.file=/etc/prometheus/prometheus-scrape-config.yaml
```
### Perguntas
- Como hospedar você mesmo no rapberry pi?
- Como hospedar na deep web?