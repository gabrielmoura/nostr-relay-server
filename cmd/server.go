package cmd

import (
	"context"
	"errors"
	"fmt"

	stdnet "net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	croncmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/cron"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	relaynet "github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/gabrielmoura/nostr-relay-server/infra/pubsub"
	redisqueue "github.com/gabrielmoura/nostr-relay-server/infra/queue/redis"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	"github.com/gabrielmoura/nostr-relay-server/internal/bootstrap"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/down"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	policies2 "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	syncjob "github.com/gabrielmoura/nostr-relay-server/internal/sync"
	"github.com/gabrielmoura/nostr-relay-server/internal/wot"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts a Nostr Relay Server",
	Long:  `Starts a Nostr Relay Server that receives messages from a Nostr Client and forwards them to a Nostr Server.`,
	Run:   runServer,
}
var bootstrapFlag bool

func runServer(cmd *cobra.Command, args []string) {

	if cmd.Flag("config").Value != nil {
		// Ler arquivo de configuração
		if err := config.LoadConfig(); err != nil {
			fmt.Println("Erro ao carregar a configuração:", err)
		}

		log.Init()
		instanceLockPath := filepath.Join("data", "nrserver.lock")
		instanceLock, err := acquireServerLock(instanceLockPath, log.Logger)
		if err != nil {
			var runningErr *serverAlreadyRunningError
			if errors.As(err, &runningErr) {
				log.Logger.Error("já existe uma instância do nrserver em execução",
					zap.Int("pid", runningErr.PID),
					zap.String("lock", runningErr.Path))
				return
			}
			log.Logger.Error("falha ao adquirir lock de instância do nrserver",
				zap.String("lock", instanceLockPath),
				zap.Error(err))
			return
		}
		defer func() {
			if err := instanceLock.Release(); err != nil {
				log.Logger.Warn("falha ao remover lock de instância do nrserver",
					zap.String("lock", instanceLock.Path()),
					zap.Error(err))
			}
		}()

		mainCtx, mainCancel := context.WithCancel(context.Background())
		var (
			in           *fiber.App
			ex           *fiber.App
			pm           *privacy.Manager
			shutdownOnce sync.Once
		)
		shutdown := func() {
			shutdownOnce.Do(func() {
				ingestion.Stop()

				if ps := pubsub.GetPubSub(); ps != nil {
					ps.Close()
				}

				shutdownFiberApp(ex, "external")
				shutdownFiberApp(in, "internal")

				if pm != nil {
					pm.Close()
				}

				mainCancel()

				if client := redis.GetClient(); client != nil {
					if err := client.Close(); err != nil {
						log.Logger.Warn("failed to close Redis client", zap.Error(err))
					}
				}
			})
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Logger.Error("server panic during startup or runtime", zap.Any("panic", recovered))
			}
			shutdown()
		}()

		// Iniciar Redis (cache + pub/sub)
		if err := redis.Init(&config.Cfg.Redis); err != nil {
			log.Logger.Warn("Redis initialization failed, continuing without Redis", zap.Error(err))
		}
		cache.Init()
		if err := pubsub.Init(); err != nil {
			log.Logger.Warn("PubSub initialization failed, continuing without pub/sub", zap.Error(err))
		}
		listener.Init()

		// Iniciar Conexão com o banco de dados
		if err := db.Init(mainCtx); err != nil {
			log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
		}
		if err := nip86.Init(db.DbQueries); err != nil {
			log.Logger.Fatal("Erro ao inicializar NIP-86", zap.Error(err))
		}
		if err := nip86.ApplyRelayMetadataOverride(mainCtx); err != nil {
			log.Logger.Fatal("Erro ao aplicar override de metadata do relay", zap.Error(err))
		}

		// Canal para capturar sinais do sistema
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, shutdownSignals()...)

		metrics.RegisterMetrics()
		metrics.RegisterSecurityMetrics()
		if config.Cfg.Jobs.Enabled {
			queueRuntime, err := redisqueue.NewRuntime(redis.GetClient(), config.Cfg.Redis.Queue, config.Cfg.Jobs)
			if err != nil {
				log.Logger.Warn("queue runtime initialization failed", zap.Error(err))
			} else {
				if err := down.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Fatal("failed to register download queue handlers", zap.Error(err))
				}
				if err := syncjob.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Fatal("failed to register sync queue handlers", zap.Error(err))
				}
				if err := internalblossom.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Fatal("failed to register blossom queue handlers", zap.Error(err))
				}
				if err := croncmd.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Fatal("failed to register cron queue handlers", zap.Error(err))
				}
				jobcore.SetDefault(queueRuntime.Service())
				if err := queueRuntime.Start(mainCtx); err != nil {
					log.Logger.Fatal("failed to start queue runtime", zap.Error(err))
				}
			}
		}
		if err := groups.Init(db.DbQueries); err != nil {
			log.Logger.Fatal("Erro ao inicializar NIP-29", zap.Error(err))
		}
		if err := security.Init(); err != nil {
			log.Logger.Fatal("Erro ao inicializar a camada de seguranca", zap.Error(err))
		}
		policies2.Init()

		// Initialize and start batch ingestion
		ingestion.Init()
		ingestion.Start(mainCtx)
		wot.Start(mainCtx)
		stream.Start(mainCtx)

		// Inicializa o handler dentro do contexto principal
		in, ex = relaynet.Router()

		// Camada de privacidade opcional (Tor / I2P / Yggdrasil)
		if config.Cfg.Privacy.Enabled {
			pm = privacy.NewManager(config.Cfg.Privacy, log.Logger)
			if err := pm.Start(mainCtx, config.Cfg.Port); err != nil {
				if config.Cfg.Privacy.Required {
					log.Logger.Error("required privacy layer failed to start", zap.Error(err))
					return
				}
				log.Logger.Error("Erro ao iniciar camada de privacidade", zap.Error(err))
			}
		}
		// Expose the manager to the admin dashboard /privacy/status handler.
		privacy.SetManager(pm)
		if err := metrics.RegisterPrivacyMetrics(prometheus.DefaultRegisterer, pm); err != nil {
			log.Logger.Error("failed to register privacy metrics", zap.Error(err))
		}

		if config.Cfg.Stream.StreamUp || config.Cfg.Stream.StreamDown {
			if err := nostrpool.Init(mainCtx, config.Cfg.Stream.Relays); err != nil {
				log.Logger.Error("Erro ao inicializar o Relay Pool", zap.Error(err))
			}
		}

		// Goroutine para aguardar sinais de desligamento
		go func() {
			<-stopChan

			log.Logger.Info("Sinal de desligamento recebido. Finalizando...")
			shutdown()
		}()

		if bootstrapFlag {
			bootstrap.CreateInitialEvents()
		}
		internalAddress := fmt.Sprintf(":%d", config.Cfg.Port+1)
		lnIn, err := prepareFiberListener(internalAddress)
		if err != nil {
			log.Logger.Error("falha ao abrir listener interno",
				zap.String("address", internalAddress),
				zap.Int("port", config.Cfg.Port+1),
				zap.Error(err))
			return
		}

		externalAddress := fmt.Sprintf(":%d", config.Cfg.Port)
		lnEx, err := prepareFiberListener(externalAddress)
		if err != nil {
			_ = lnIn.Close()
			log.Logger.Error("falha ao abrir listener externo",
				zap.String("address", externalAddress),
				zap.Int("port", config.Cfg.Port),
				zap.Error(err))
			return
		}

		startFiberListener(mainCtx, in, lnIn, "internal", internalAddress)
		startFiberListener(mainCtx, ex, lnEx, "external", externalAddress)

		// Aguarda pelo término do contexto principal
		<-mainCtx.Done()

		log.Logger.Info("Servidor finalizado com sucesso.")
	}
}

func prepareFiberListener(address string) (stdnet.Listener, error) {
	listener, err := relaynet.PrepareListen(address)
	if err != nil {
		return nil, fmt.Errorf("prepare listener: %w", err)
	}
	return listener, nil
}

func startFiberListener(ctx context.Context, app *fiber.App, listener stdnet.Listener, name, address string) {
	go func() {
		if err := app.Listener(listener); err != nil {
			if ctx.Err() != nil {
				log.Logger.Debug("Fiber listener stopped during shutdown",
					zap.String("server", name),
					zap.String("address", address),
					zap.Error(err))
				return
			}
			log.Logger.Error("Fiber listener stopped unexpectedly",
				zap.String("server", name),
				zap.String("address", address),
				zap.Error(err))
		}
	}()
}

func shutdownFiberApp(app *fiber.App, name string) {
	if app == nil {
		return
	}
	if err := app.Shutdown(); err != nil {
		log.Logger.Warn("failed to shut down Fiber server",
			zap.String("server", name),
			zap.Error(err))
	}
}

func init() {
	serverCmd.Flags().BoolP("config", "c", true, "Enable configuration file")
	serverCmd.Flags().BoolVarP(&bootstrapFlag, "bootstrap", "b", false, "Enable bootstrap")
	rootCmd.AddCommand(serverCmd)
}
