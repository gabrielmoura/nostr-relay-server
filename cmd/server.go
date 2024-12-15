package cmd

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/spf13/cobra"
	"log"

	"os"
	"os/signal"
	"syscall"

	slog "github.com/gabrielmoura/nostr-relay-server/infra/log"
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
			log.Fatalf("Erro ao carregar a configuração: %v", err)
		}

		slog.Init()

		mainCtx, mainCancel := context.WithCancel(context.Background())
		defer mainCancel()

		// Iniciar Conexão com o banco de dados
		if err := db.Init(mainCtx); err != nil {
			log.Fatalf("Erro ao iniciar conexão com o banco de dados: %v", err)
		}

		// Canal para capturar sinais do sistema
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

		// Inicializa o handler dentro do contexto principal
		server := handler.Init(mainCtx)

		// Goroutine para aguardar sinais de desligamento
		go func() {
			<-stopChan

			log.Print("Sinal de desligamento recebido. Finalizando...")

			mainCancel()

			// Chamar o método Shutdown do servidor
			if err := server.Shutdown(mainCtx); err != nil {
				log.Fatalf("Erro ao desligar o servidor: %s", err.Error())
			}
		}()

		ln, _ := net.PrepareListen(server)

		// Iniciar o servidor HTTP
		server.Serve(ln)

		// Aguarda pelo término do contexto principal
		<-mainCtx.Done()

		log.Println("Servidor finalizado com sucesso.")
	}
}

func init() {
	serverCmd.Flags().BoolP("config", "c", true, "Enable configuration file")
	rootCmd.AddCommand(serverCmd)
}
