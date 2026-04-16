# negentropy_go

Biblioteca Go para processamento de Negentropy (NIP-77) sem acoplamento com WebSocket, HTTP ou banco concreto.

## Objetivo

- Processar payloads `NEG-OPEN` e `NEG-MSG` em hexadecimal.
- Delegar busca de eventos para interfaces (`EventStore`).
- Reaproveitar resultados de busca com cache e deduplicacao de consultas concorrentes.

## Pacotes

- `model`: tipos de entrada/saida e filtros.
- `contracts`: contratos para storage e cache.
- `cache`: implementacao de cache em memoria.
- `protocol`: codec do protocolo binario negentropy.
- `engine`: reconciliador puro do algoritmo.
- `service`: gerenciamento de sessoes e orquestracao.

## Exemplo

Veja `examples/inmemory/main.go`.
