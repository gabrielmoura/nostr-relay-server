package cmd

import (
	"context"
	"fmt"
	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/eventstore/slicestore"
	"github.com/gabrielmoura/nostr-relay-server/config"
	zap "github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77"
	"github.com/spf13/cobra"
	"log"
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
		log.Fatalf("Erro ao carregar a configuração: %v", err)
	}

	zap.Init()

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	// Iniciar Conexão com o banco de dados
	if err := db.Init(mainCtx); err != nil {
		log.Fatalf("Erro ao iniciar conexão com o banco de dados: %v", err)
	}

	// Canal para capturar sinais do sistema
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	remote := cmd.Flag("remote").Value.String()
	if err := validateRemote(remote); err != nil {
		log.Fatalf("Remote: %v", err)
	}
	pk := cmd.Flag("pk").Value.String()
	if pk == "" {
		log.Fatalf("Public Key is required")
	}

	dbx := &slicestore.SliceStore{}
	err := dbx.Init()
	if err != nil {
		return
	}
	local := eventstore.RelayWrapper{Store: dbx}

	// Goroutine para aguardar sinais de desligamento
	go func() {
		<-stopChan

		log.Print("Sinal de desligamento recebido. Finalizando...")

		mainCancel()
	}()

	err = nip77.NegentropySync(mainCtx,
		local, remote, nostr.Filter{
			Authors: []string{pk},
		})
	if err != nil {
		panic(err)
	}

	data, err := local.QuerySync(mainCtx, nostr.Filter{
		Authors: []string{pk},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("total events:", len(data))

	for _, evt := range data {
		err := db.DbQueries.InsertEvent(mainCtx, evt)
		if err != nil {
			log.Fatalf("Erro ao inserir evento: %v", err)
			return
		}
	}

}
func init() {
	syncCmd.Flags().StringP("remote", "r", "", "Remote Nostr Server")
	syncCmd.Flags().StringP("pk", "p", "", "Public Key")
	rootCmd.AddCommand(syncCmd)
}

func validateRemote(remote string) error {
	if remote == "" {
		return fmt.Errorf("Remote is required")
	}
	if remote[:5] != "ws://" && remote[:6] != "wss://" {
		return fmt.Errorf("Remote must start with ws:// or wss://")
	}
	return nil
}
