package cmd

import (
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/internal/bootstrap"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	policies2 "github.com/gabrielmoura/nostr-relay-server/internal/policies"
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

		// Iniciar Conexão com o banco de dados
		if err := db.Init(mainCtx); err != nil {
			log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
		}

		// Canal para capturar sinais do sistema
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

		metrics.RegisterMetrics()
		policies2.Init()
		// Inicializa o handler dentro do contexto principal
		in, ex := net.Router()

		if config.Cfg.Stream.StreamUp || config.Cfg.Stream.StreamDown {
			// TODO: Remover este bloco de código daqui
			if err := nostrpool.Init(mainCtx, config.Cfg.Stream.Relays); err != nil {
				log.Logger.Error("Erro ao inicializar o Relay Pool", zap.Error(err))
			}
		}

		// Goroutine para aguardar sinais de desligamento
		go func() {
			<-stopChan

			log.Logger.Info("Sinal de desligamento recebido. Finalizando...")

			mainCancel()

			// Chamar o método Shutdown do servidor
			if err := ex.Shutdown(); err != nil {
				log.Logger.Fatal("Erro ao desligar o servidor", zap.Error(err))
			}
			if err := in.Shutdown(); err != nil {
				log.Logger.Fatal("Erro ao desligar o servidor", zap.Error(err))
			}

		}()

		if bootstrapFlag {
			bootstrap.CreateInitialEvents()
		}
		lnIn, _ := net.PrepareListen(":9091")
		lnEx, _ := net.PrepareListen(":9090")

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
