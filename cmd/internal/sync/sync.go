package sync

import (
	"context"
	"fmt"
	"github.com/fiatjaf/eventstore"
	dbstore "github.com/gabrielmoura/nostr-relay-server/cmd/internal/sync/store"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/nip77"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func Sync(cf *ConfSync) {
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Erro ao carregar a configuração: %v", err)
	}

	log.Init()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handleGracefulShutdown(cancel)

	if err := validateRemote(cf.Remote); err != nil {
		log.Logger.Fatal("Remote inválido", zap.Error(err))
	}

	filter := nostr.Filter{}
	if cf.Pk != "" {
		if strings.HasPrefix(cf.Pk, "npub") {
			_, npk, err := nip19.Decode(cf.Pk)
			if err != nil {
				log.Logger.Fatal("Chave pública inválida", zap.Error(err))
				return
			}
			cf.Pk = npk.(string)
		}
		filter.Authors = []string{cf.Pk}
	}

	store := dbstore.NewDBStore(ctx)
	local := eventstore.RelayWrapper{Store: store}

	go func() {
		if err := syncEvents(ctx, local, cf); err != nil {
			log.Logger.Fatal("Erro ao sincronizar", zap.Error(err))
		}
		cancel()
	}()

	<-ctx.Done()
	log.Logger.Info("Finalizado")
}

func handleGracefulShutdown(cancel context.CancelFunc) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopChan
		log.Logger.Info("Sinal de desligamento recebido. Finalizando...")
		cancel()
	}()
}

func validateRemote(remote string) error {
	if remote == "" {
		return fmt.Errorf("remote is required")
	}
	if !(strings.HasPrefix(remote, "ws://") || strings.HasPrefix(remote, "wss://")) {
		return fmt.Errorf("remote must start with ws:// or wss://")
	}
	return nil
}

func syncEvents(ctx context.Context, local eventstore.RelayWrapper, cf *ConfSync) error {
	filter := nostr.Filter{}
	if cf.Pk != "" {
		filter.Authors = []string{cf.Pk}
	}

	if err := nip77.NegentropySync(ctx, local, cf.Remote, filter, cf.Direction); err != nil {
		return err
	}
	log.Logger.Info("Sincronização finalizada.")
	return nil
}
