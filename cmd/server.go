package cmd

import (
	"context"
	"errors"
	"fmt"

	"io"
	stdnet "net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	croncmd "github.com/gabrielmoura/nostr-relay-server/cmd/internal/cron"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	httpblossom "github.com/gabrielmoura/nostr-relay-server/infra/handler/store/blossom"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	relaynet "github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/gabrielmoura/nostr-relay-server/infra/pubsub"
	redisqueue "github.com/gabrielmoura/nostr-relay-server/infra/queue/redis"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
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
			lnIn         stdnet.Listener
			lnEx         stdnet.Listener
			pprofServer  io.Closer
			shutdownOnce sync.Once
		)
		shutdown := func() {
			shutdownOnce.Do(func() {
				// Cancel first so every long-running component, including the Fiber
				// listener goroutines, can classify its close as an expected shutdown.
				mainCancel()

				ingestion.Stop()

				if ps := pubsub.GetPubSub(); ps != nil {
					ps.Close()
				}

				shutdownFiberApp(ex, "external")
				shutdownFiberApp(in, "internal")
				closeFiberListener(lnEx, "external")
				closeFiberListener(lnIn, "internal")
				closeDevelopmentPprof(pprofServer)

				if pm != nil {
					pm.Close()
				}

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

		// Bind both sockets before initializing any dependency. Keeping these
		// listeners open through startup prevents another process from taking a
		// relay port between validation and Fiber.Listener.
		in, ex = relaynet.Router()
		internalAddress := fmt.Sprintf(":%d", config.Cfg.Port+1)
		externalAddress := fmt.Sprintf(":%d", config.Cfg.Port)
		lnIn, lnEx, err = prepareFiberListeners(internalAddress, externalAddress)
		if err != nil {
			log.Logger.Error("falha ao abrir listeners do relay",
				zap.String("internal_address", internalAddress),
				zap.String("external_address", externalAddress),
				zap.Error(err))
			return
		}
		blobStore, err := blobstore.NewConfiguredStore(mainCtx, config.Cfg.Store, "files")
		if err != nil {
			log.Logger.Error("failed to initialize Blossom blob storage",
				zap.String("backend", config.Cfg.Store.NormalizedBackend()),
				zap.Error(err))
			return
		}
		httpblossom.SetStore(blobStore)
		pprofServer = startDevelopmentPprof(config.Cfg.AppEnv, log.Logger)

		// Iniciar Redis (cache + pub/sub)
		if err := redis.Init(&config.Cfg.Redis); err != nil {
			log.Logger.Warn("Redis initialization failed, continuing without Redis", zap.Error(err))
		}
		if err := pubsub.Init(); err != nil {
			log.Logger.Warn("PubSub initialization failed, continuing without pub/sub", zap.Error(err))
		}
		listener.Init()

		// Iniciar Conexão com o banco de dados
		if err := db.Init(mainCtx); err != nil {
			log.Logger.Error("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
			return
		}
		if err := nip86.Init(db.DbQueries); err != nil {
			log.Logger.Error("Erro ao inicializar NIP-86", zap.Error(err))
			return
		}
		if err := nip86.ApplyRelayMetadataOverride(mainCtx); err != nil {
			log.Logger.Error("Erro ao aplicar override de metadata do relay", zap.Error(err))
			return
		}

		// Canal para capturar sinais do sistema
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, shutdownSignals()...)
		defer signal.Stop(stopChan)

		metrics.RegisterMetrics()
		metrics.RegisterSecurityMetrics()
		if config.Cfg.Jobs.Enabled {
			queueRuntime, err := redisqueue.NewRuntime(redis.GetClient(), config.Cfg.Redis.Queue, config.Cfg.Jobs)
			if err != nil {
				log.Logger.Warn("queue runtime initialization failed", zap.Error(err))
			} else {
				if err := down.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Error("failed to register download queue handlers", zap.Error(err))
					return
				}
				if err := syncjob.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Error("failed to register sync queue handlers", zap.Error(err))
					return
				}
				if err := internalblossom.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Error("failed to register blossom queue handlers", zap.Error(err))
					return
				}
				if err := croncmd.RegisterQueueHandlers(queueRuntime.Registry()); err != nil {
					log.Logger.Error("failed to register cron queue handlers", zap.Error(err))
					return
				}
				jobcore.SetDefault(queueRuntime.Service())
				if err := queueRuntime.Start(mainCtx); err != nil {
					log.Logger.Error("failed to start queue runtime", zap.Error(err))
					return
				}
			}
		}
		if err := groups.Init(db.DbQueries); err != nil {
			log.Logger.Error("Erro ao inicializar NIP-29", zap.Error(err))
			return
		}
		if err := security.Init(); err != nil {
			log.Logger.Error("Erro ao inicializar a camada de seguranca", zap.Error(err))
			return
		}
		policies2.Init()

		// Initialize and start batch ingestion
		ingestion.Init()
		ingestion.Start(mainCtx)
		wot.Start(mainCtx)
		stream.Start(mainCtx)

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
			if pm.Status().Degraded {
				log.Logger.Warn("privacy layer degraded; relay will continue in fail-open mode",
					zap.Bool("required", config.Cfg.Privacy.Required))
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
			select {
			case <-mainCtx.Done():
				return
			case sig := <-stopChan:
				log.Logger.Info("Sinal de desligamento recebido. Finalizando...", zap.String("signal", sig.String()))
				shutdown()
			}
		}()

		if bootstrapFlag {
			bootstrap.CreateInitialEvents()
		}
		startFiberListener(mainCtx, in, lnIn, "internal", internalAddress, shutdown)
		startFiberListener(mainCtx, ex, lnEx, "external", externalAddress, shutdown)

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

func prepareFiberListeners(internalAddress, externalAddress string) (stdnet.Listener, stdnet.Listener, error) {
	internalListener, err := prepareFiberListener(internalAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare internal listener: %w", err)
	}

	externalListener, err := prepareFiberListener(externalAddress)
	if err != nil {
		if closeErr := internalListener.Close(); closeErr != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("prepare external listener: %w", err),
				fmt.Errorf("close internal listener after external bind failure: %w", closeErr),
			)
		}
		return nil, nil, fmt.Errorf("prepare external listener: %w", err)
	}

	return internalListener, externalListener, nil
}

func startFiberListener(ctx context.Context, app *fiber.App, listener stdnet.Listener, name, address string, shutdown func()) {
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
			shutdown()
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

func closeFiberListener(listener stdnet.Listener, name string) {
	if listener == nil {
		return
	}
	if err := listener.Close(); err != nil && !errors.Is(err, stdnet.ErrClosed) {
		log.Logger.Warn("failed to close Fiber listener",
			zap.String("server", name),
			zap.Error(err))
	}
}

func init() {
	serverCmd.Flags().BoolP("config", "c", true, "Enable configuration file")
	serverCmd.Flags().BoolVarP(&bootstrapFlag, "bootstrap", "b", false, "Enable bootstrap")
	rootCmd.AddCommand(serverCmd)
}
