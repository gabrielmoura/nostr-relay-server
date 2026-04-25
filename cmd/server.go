package cmd

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/infra/pubsub"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/bootstrap"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	policies2 "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/gabrielmoura/nostr-relay-server/internal/wot"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"

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
		mainCtx, mainCancel := context.WithCancel(context.Background())
		defer mainCancel()

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

		// Inicializar prepared statements (apenas em produção)
		if config.Cfg.AppEnv == "production" {
			if err := db.InitPreparedStatements(mainCtx, db.Pool); err != nil {
				log.Logger.Warn("Prepared statements initialization failed, continuing without them", zap.Error(err))
			}
		}

		// Canal para capturar sinais do sistema
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

		metrics.RegisterMetrics()
		metrics.RegisterSecurityMetrics()
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
		in, ex := net.Router()

		if config.Cfg.Stream.StreamUp || config.Cfg.Stream.StreamDown {
			if err := nostrpool.Init(mainCtx, config.Cfg.Stream.Relays); err != nil {
				log.Logger.Error("Erro ao inicializar o Relay Pool", zap.Error(err))
			}
		}

		// Goroutine para aguardar sinais de desligamento
		go func() {
			<-stopChan

			log.Logger.Info("Sinal de desligamento recebido. Finalizando...")

			mainCancel()

			// Shutdown ingestion
			ingestion.Stop()

			// Shutdown pubsub
			if ps := pubsub.GetPubSub(); ps != nil {
				ps.Close()
			}

			// Chamar o método Shutdown do servidor
			if err := ex.Shutdown(); err != nil {
				log.Logger.Fatal("Erro ao desligar o servidor", zap.Error(err))
			}
			if err := in.Shutdown(); err != nil {
				log.Logger.Fatal("Erro ao desligar o servidor", zap.Error(err))
			}

			// Fechar conexão Redis
			if client := redis.GetClient(); client != nil {
				client.Close()
			}
		}()

		if bootstrapFlag {
			bootstrap.CreateInitialEvents()
		}
		lnIn, _ := net.PrepareListen(fmt.Sprintf(":%d", config.Cfg.Port+1))
		lnEx, _ := net.PrepareListen(fmt.Sprintf(":%d", config.Cfg.Port))

		go in.Listener(lnIn)
		go ex.Listener(lnEx)

		// Aguarda pelo término do contexto principal
		<-mainCtx.Done()

		log.Logger.Info("Servidor finalizado com sucesso.")
	}
}

func init() {
	serverCmd.Flags().BoolP("config", "c", true, "Enable configuration file")
	serverCmd.Flags().BoolVarP(&bootstrapFlag, "bootstrap", "b", false, "Enable bootstrap")
	rootCmd.AddCommand(serverCmd)
}
