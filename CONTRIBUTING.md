Crie uma Issue para Erros e caso seja um pedido de funcionalidade use o prefixo feature.
Seja educado, descreva ao máximo o problema use logs se possível para ser possível repoduzir o problema e resolver o
ocorrido.

## Database migrations

Schema changes live in `infra/db/migrations/` as paired `.up.sql` and
`.down.sql` files. Do not edit an existing migration after it has been merged.
Create a new pair with `nrserver migrate create <name>`, review both SQL files,
then validate the complete roundtrip against a disposable PostgreSQL database:

```bash
nrserver migrate up
nrserver migrate down --all
nrserver migrate up
```

Before deploying a binary, run `nrserver migrate up` against its target
database. The server validates the applied version at boot and refuses to start
when the schema is outdated or dirty.


*   [Documento de informações do relé](https://github.com/nostr-protocol/nips/blob/master/11.md)
*   [Descrição básica do fluxo do protocolo](https://github.com/nostr-protocol/nips/blob/master/01.md)
*   [Lista de seguidores](https://github.com/nostr-protocol/nips/blob/master/02.md)
*   [Mensagem direta criptografada](https://github.com/nostr-protocol/nips/blob/master/04.md)
*   [Reações](https://github.com/nostr-protocol/nips/blob/master/25.md)
*   [Solicitação de exclusão de evento](https://github.com/nostr-protocol/nips/blob/master/09.md)
*   [Conteúdo de formato longo](https://github.com/nostr-protocol/nips/blob/master/23.md)
*   [Resultados da contagem](https://github.com/nostr-protocol/nips/blob/master/45.md)
*   [Autenticação de clientes para retransmissores](https://github.com/nostr-protocol/nips/blob/master/42.md)
*   [Mensagens diretas privadas](https://github.com/nostr-protocol/nips/blob/master/17.md)
*   [Desconhecido NIP-62](https://github.com/nostr-protocol/nips/blob/master/62.md)
*   [Carimbo de data e hora de expiração](https://github.com/nostr-protocol/nips/blob/master/40.md)
*   [Repostagens](https://github.com/nostr-protocol/nips/blob/master/18.md)
*   [Prova de trabalho](https://github.com/nostr-protocol/nips/blob/master/13.md)
