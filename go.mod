module github.com/gabrielmoura/nostr-relay-server

go 1.26.3

require (
	github.com/99designs/gqlgen v0.17.90
	github.com/buckket/go-blurhash v1.1.0
	github.com/bytedance/sonic v1.15.0
	github.com/coder/websocket v1.8.15
	github.com/disintegration/imaging v1.6.2
	github.com/emmansun/base64 v0.9.0
	github.com/fasthttp/websocket v1.5.12
	github.com/gofiber/contrib/websocket v1.3.4
	github.com/gofiber/fiber/v2 v2.52.12
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.1
	github.com/minio/sha256-simd v1.0.1
	github.com/nbd-wtf/go-nostr v0.52.3
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/redis/go-redis/v9 v9.18.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc
	github.com/valyala/fasthttp v1.70.0
	github.com/vektah/gqlparser/v2 v2.5.33
	go.uber.org/zap v1.27.1
	golang.org/x/time v0.15.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Arceliar/ironwood v0.0.0-20260613025018-d50055b11f5e // indirect
	github.com/Arceliar/phony v0.0.0-20220903101357-530938a4b13d // indirect
	github.com/ImVexed/fasturl v0.0.0-20230304231329-4e41488060f3 // indirect
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beevik/ntp v1.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.5 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.7.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.6 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.5 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0 // indirect
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic/loader v0.5.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/bubbles v1.0.0 // indirect
	github.com/charmbracelet/bubbletea v1.3.10 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.11.7 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/cretz/bine v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-i2p/common v0.1.59999 // indirect
	github.com/go-i2p/crypto v0.1.59999 // indirect
	github.com/go-i2p/elgamal v0.1.59999 // indirect
	github.com/go-i2p/go-i2p v0.1.59999 // indirect
	github.com/go-i2p/go-i2pcontrol v0.1.9-0.20260607233455-950087a3858f // indirect
	github.com/go-i2p/go-nat-listener v0.0.0-20260402222111-bfda0025cb1b // indirect
	github.com/go-i2p/go-noise v0.1.59999 // indirect
	github.com/go-i2p/go-unzip v0.0.0-20260417162122-21146ed7aca8 // indirect
	github.com/go-i2p/i2p-control v0.0.0-20260608000308-d8ed861a259c // indirect
	github.com/go-i2p/i2ptui v0.0.0-20260607232722-b95832d1b84a // indirect
	github.com/go-i2p/logger v0.1.59999 // indirect
	github.com/go-i2p/noise v1.1.1-0.20260327201800-8e41bb3d9f1e // indirect
	github.com/go-i2p/path v0.1.59999 // indirect
	github.com/go-i2p/pool v0.1.5999 // indirect
	github.com/go-i2p/red25519 v0.0.0-20260302212615-1093a31f680d // indirect
	github.com/go-i2p/su3 v0.1.59999 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/gologme/log v1.3.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hjson/hjson-go/v4 v4.6.0 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mailru/easyjson v0.9.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/quic-go/quic-go v0.60.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/oops v1.22.0 // indirect
	github.com/savsgio/gotils v0.0.0-20250924091648-bce9a52d7761 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sosodev/duration v1.4.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/things-go/go-socks5 v0.1.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/urfave/cli/v3 v3.8.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/voluminor/ratatoskr v1.1.0 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/ybbus/jsonrpc/v2 v2.1.7 // indirect
	github.com/yggdrasil-network/yggdrasil-go v0.5.14 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.step.sm/crypto v0.82.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/arch v0.26.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/image v0.0.0-20191009234506-e7c1f5e7dbb8 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gvisor.dev/gvisor v0.0.0-20250812171554-968e93457fe6 // indirect
)

tool github.com/99designs/gqlgen
