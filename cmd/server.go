package cmd

import (
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
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
		// Inicializa o handler dentro do contexto principal
		server := handler.Init(mainCtx)

		// Goroutine para aguardar sinais de desligamento
		go func() {
			<-stopChan

			log.Logger.Info("Sinal de desligamento recebido. Finalizando...")

			mainCancel()

			// Chamar o método Shutdown do servidor
			if err := server.Shutdown(mainCtx); err != nil {
				log.Logger.Fatal("Erro ao desligar o servidor", zap.Error(err))
			}
		}()

		ln, _ := net.PrepareListen(server)

		// Iniciar o servidor HTTP
		server.Serve(ln)

		// Aguarda pelo término do contexto principal
		<-mainCtx.Done()

		log.Logger.Info("Servidor finalizado com sucesso.")
	}
}

func init() {
	serverCmd.Flags().BoolP("config", "c", true, "Enable configuration file")
	rootCmd.AddCommand(serverCmd)
}
