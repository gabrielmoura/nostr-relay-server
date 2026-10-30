# RelaySync v2: avaliação de storages Negentropy

Status: levantamento aprovado como entrada da Fase 2; benchmarks ainda
pendentes. Esta decisão trata do storage interno do reconciliador, não substitui
`SnapshotProvider`, `JournalProvider` ou o banco de eventos do relay.

## Fontes analisadas

O projeto oficial do Negentropy lista a implementação C++ de referência como
estável e seus storages `Vector`, `BTreeMem`, `BTreeLMDB` e `SubRange`. A mesma
fonte explica que a suíte de conformidade executa fuzz tests entre linguagens,
compara saídas byte a byte e verifica downgrade de versão.

Também foi analisado `HORNET-Storage/go-negentropy`. Ele é uma implementação Go
pequena, declara compatibilidade com a suíte do projeto oficial e expõe uma
abstração `Storage` com operações de tamanho, item, iteração, lower-bound e
fingerprint. No estado consultado, oferece `Vector`; não oferece as variantes
B-tree da referência C++.

Referências: [implementations e conformance do projeto oficial](https://github.com/hoytech/negentropy#implementations),
[storage C++](https://github.com/hoytech/negentropy/blob/master/cpp/README.md),
[HORNET-Storage/go-negentropy](https://github.com/HORNET-Storage/go-negentropy),
[API Go publicada](https://pkg.go.dev/github.com/illuzen/go-negentropy).

## O que Negentropy exige do storage

O algoritmo trabalha sobre registros ordenados por `(timestamp, id)` e precisa
de:

- lower-bound por limite `(timestamp, prefixo de ID)`;
- iteração de uma faixa;
- fingerprint de uma faixa;
- leitura de item/ID para produzir `IdList`;
- visão estável durante todos os rounds da sessão;
- limite de frame e capacidade de repetir diferenças descobertas.

O protocolo permite estruturas mais avançadas que arrays, mas não define uma
árvore específica. O storage é uma otimização interna; ele não altera o wire
Negentropy V1 (`0x61`).

## Comparação

| Storage | Pontos fortes | Custos e riscos | Adequação ao RelaySync |
| --- | --- | --- | --- |
| `Vector` | Contíguo, simples, baixo overhead, bom cache locality, construção rápida para query dinâmica e fácil serialização de snapshot | Inserção/remoção linear; fingerprint ingênuo pode varrer a faixa; exige selagem para estabilidade | **Melhor base para partições imutáveis** |
| `BTreeMem` | Inserção, remoção e fingerprint logarítmicos; acompanha dataset mutável | Mais overhead por nó; não persistente; referência C++ declara que não é thread-safe; snapshot concorrente exige coordenação externa | Alternativa para índice mutável especializado, não default |
| `BTreeLMDB` | Persistência, transações ACID, acesso concorrente e fingerprints agregados na árvore; page cache reduz diferença de leitura | Duplica o storage do relay; introduz LMDB, sizing de mapas, locks, backup/recovery e operação adicional; não é agnóstico de banco | Não adequado ao adapter inicial PostgreSQL; candidato opcional futuro |
| `SubRange` | Proxy para uma faixa de outro storage; permite reconciliação parcial sem copiar a árvore | Não é backend; limites e lifetime precisam permanecer válidos; não resolve persistência nem concorrência | **Abstração ideal para cada partição/snapshot** |

As características de `BTreeMem` e `BTreeLMDB` acima são as da implementação
C++ de referência. Não devem ser atribuídas automaticamente ao package Go.

## Decisão para o RelaySync

O reconciliador usará uma composição lógica:

```text
SnapshotProvider
    -> PartitionReader ordenado e imutável
        -> SubRange lógico (limites da partição)
            -> Vector selado de EventMetadata
                -> prefix sums/contagens para fingerprints O(1) por faixa
```

O `Vector` não representa o relay inteiro. Ele representa somente uma partição
divergente, materializada durante a sessão a partir de metadados compactos. O
payload completo dos eventos fica no snapshot/event store e não no storage do
Negentropy.

### Por que essa composição se encaixa

1. Snapshots são imutáveis: a desvantagem de inserir/remover do `Vector` não
   aparece durante os rounds.
2. A partição limita o pico de RAM. O perfil inicial terá `max_partition_refs`
   finito; partições acima desse limite são subdivididas deterministicamente.
3. O `SubRange` lógico impede que o reconciliador enxergue dados fora da
   partição autorizada e evita copiar um índice global.
4. Metadados compactos (`created_at` e Event ID) são suficientes para
   Negentropy; conteúdos e tags não entram na RAM do reconciliador.
5. Prefix sums de IDs, contagem e lower-bound sobre slice ordenado evitam
   recalcular fingerprints varrendo milhões de itens em cada round.
6. A implementação é independente dos pacotes analisados e pode ser extraída
   junto com `pkg/relaySync`.

## Fingerprint incremental

O fingerprint V1 é calculado sobre a soma modular dos IDs, a contagem da faixa,
SHA-256 e truncamento definido pela especificação. Para uma `Vector` imutável,
o adapter poderá manter:

- `ids[]`/metadata ordenado;
- prefix count;
- prefix sum de 256 bits em representação little-endian compatível;
- índice opcional de bounds.

Uma faixa `[lo, hi)` obtém soma e contagem por diferença de prefixos e executa
somente o hash final. O custo adicional de prefix sums é linear na partição e
fica sujeito ao limite de memória. Se o benchmark mostrar que esse custo é
alto, a alternativa é um acumulador em árvore por partição, sem mudar a API ou
o wire.

O digest do Manifest continua separado: ele usa o formato canônico RelaySync
descrito em `architecture.md`; não reutiliza bytes de fingerprint Negentropy nem
serialização Protobuf.

## Memória e escala

O perfil inicial não carregará 10 milhões de eventos completos. Ele carrega, no
máximo, os metadados da partição corrente, seus agregados e buffers de frame.
Cada referência requer Event ID de 32 bytes, timestamp e overhead controlado;
payload, tags e conteúdo ficam fora desse heap.

O limite operacional será medido, não presumido:

- quantidade máxima de refs por partição;
- bytes de metadata por ref;
- bytes de prefix aggregate;
- frame Negentropy;
- número máximo de rounds;
- concorrência de partições por sessão.

Uma sessão não reconcilia todas as partições em paralelo por padrão. O scheduler
usa limite de workers e backpressure; partições são liberadas antes da próxima
quando o protocolo permitir.

## Quando usar cada alternativa

### Vector

Usar para snapshot materializado, partição histórica imutável, query dinâmica
pequena/média, testes, golden vectors e conformance harness. Não usar como um
vetor global do relay com dezenas de milhões de referências.

### BTreeMem

Considerar se um consumidor independente possuir um índice de metadados mutável
que precise refletir inserts/deletes continuamente. Mesmo nesse caso, o
RelaySync deve abrir uma visão consistente ou usar copy-on-write; não ler uma
árvore mutável diretamente enquanto os rounds avançam. A implementação deve
fornecer a mesma porta de storage e testes de concorrência do Vector.

### BTreeLMDB

Considerar somente quando o relay consumidor já usar LMDB como fonte autoritativa
ou quando um benchmark demonstrar que o custo de materialização é dominante.
Não será adicionado ao NRServer PostgreSQL apenas para acelerar Negentropy:
isso duplicaria dados, criaria uma segunda recuperação e ampliaria o escopo de
operação. Uma futura integração LMDB precisa definir mapa máximo, compactação,
backup, locks, crash recovery e retenção.

### SubRange

Adotar como conceito obrigatório, possivelmente sem um tipo público chamado
`SubRange`: `PartitionReader` recebe lower/upper bounds e restringe todas as
operações. A implementação pode encapsular uma vector ou uma árvore sem que o
engine saiba qual backend existe.

## Avaliação de HORNET-Storage/go-negentropy

O package é útil para entender uma decomposição mínima de storage e para
comparar comportamento Go, especialmente o contrato `Storage` e a integração
com o harness. Ele não resolve sozinho os requisitos do RelaySync porque:

- o storage publicado é Vector;
- não fornece persistência/snapshot/journal;
- não fornece particionamento de relay, autorização ou backpressure;
- a implementação é pequena e não substitui a validação independente do core.

Não haverá dependência direta nem cópia de código. O RelaySync implementará a
porta necessária a partir da especificação pública e comparará os bytes com a
suíte oficial e com implementações de referência.

## Plano de validação

Antes de congelar o adapter:

1. comparar Vector sem prefix sums versus Vector com prefix sums;
2. medir BTreeMem apenas como baseline experimental;
3. executar conformance C++/Go independente nos quatro cenários oficiais;
4. testar partições de 100 mil, 1 milhão e, sob demanda, 10 milhões de refs;
5. medir divergências 0%, 0,1%, 1%, 10%, 50% e 100%;
6. coletar RSS, heap, allocations, CPU, rounds, bytes e duração;
7. executar race detector, fuzzing de bounds/varints e cancelamento;
8. testar snapshot estável enquanto o storage original recebe mudanças.

Critério de aceite: Vector particionado com agregados atende limites de memória
e throughput acordados sem divergência de wire. Caso contrário, selecionar uma
árvore agregada por partição com a mesma porta, sem migrar automaticamente o
storage autoritativo.

