package cmd

import (
	"context"
	"fmt"
	"github.com/fiatjaf/eventstore"
	"github.com/gabrielmoura/nostr-relay-server/infra/nip77"
	"github.com/gabrielmoura/nostr-relay-server/infra/store"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Syncs the local database with the remote database",
	Run:   runSync,
}

func runSync(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Erro ao carregar a configuração: %v", err)
	}

	log.Init()

	mainCtx, mainCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mainCancel()

	// Iniciar Conexão com o banco de dados
	if err := db.Init(mainCtx); err != nil {

		log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
	}

	// Canal para capturar sinais do sistema
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	remote := cmd.Flag("remote").Value.String()
	if err := validateRemote(remote); err != nil {
		log.Logger.Fatal("Remote", zap.Error(err))
	}
	pk := cmd.Flag("pk").Value.String()
	if pk == "" {
		log.Logger.Fatal("Public Key is required")
	}
	direction := cmd.Flag("direction").Value.String()
	filter := nostr.Filter{
		Authors: []string{pk},
	}

	data, _ := db.DbQueries.QueryEvents(mainCtx, filter)

	dbx := &store.SliceStore{
		MaxLimit: 999999,
		Events:   data,
	}
	err := dbx.Init()
	if err != nil {
		return
	}
	local := eventstore.RelayWrapper{Store: dbx}

	// Goroutine para aguardar sinais de desligamento
	go func() {
		<-stopChan

		log.Logger.Info("Sinal de desligamento recebido. Finalizando...")

		mainCancel()
	}()

	go func() {
		err = nip77.NegentropySync(mainCtx, local, remote, filter, direction)
		if err != nil {
			log.Logger.Fatal("Erro ao sincronizar", zap.Error(err))
		}

		data, err = local.QuerySync(mainCtx, filter)
		if err != nil {
			log.Logger.Fatal("Erro ao consultar eventos", zap.Error(err))
		}

		log.Logger.Info("Total de eventos", zap.Int("total", len(data)))

		for _, evt := range data {
			err := db.DbQueries.InsertEvent(mainCtx, evt)
			if err != nil {
				log.Logger.Fatal("Erro ao inserir evento", zap.Error(err))
				return
			}
		}
	}()
	<-mainCtx.Done()
	log.Logger.Info("Finalizado")

}
func init() {
	syncCmd.Flags().StringP("remote", "r", "", "Remote Nostr Server")
	syncCmd.Flags().StringP("pk", "p", "", "Public Key")
	syncCmd.Flags().StringP("direction", "d", "both", "Direction of the sync (up, down, both)")
	rootCmd.AddCommand(syncCmd)
}

func validateRemote(remote string) error {
	if remote == "" {
		return fmt.Errorf("remote is required")
	}
	if remote[:5] != "ws://" && remote[:6] != "wss://" {
		return fmt.Errorf("remote must start with ws:// or wss://")
	}
	return nil
}
