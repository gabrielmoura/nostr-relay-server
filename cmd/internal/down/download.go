package down

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"go.uber.org/zap"
)

type DownloadOptions struct {
	PublicKey string
	RelayURL  []string
	Kinds     []int
	Tags      []string
	Mentioned bool
}

const pageSize = 500

func Download(cf *DownloadOptions) {
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Erro ao carregar a configuração: %v", err)
	}

	log.Init()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := db.Init(ctx); err != nil {
		log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
	}

	if cf.PublicKey != "" {
		var err error
		cf.PublicKey, err = normalizePublicKey(cf.PublicKey)
		if err != nil {
			log.Logger.Fatal("Chave pública inválida", zap.Error(err))
			return
		}
	}

	wg := sync.WaitGroup{}

	for _, url := range cf.RelayURL {
		wg.Add(1)
		go func() {
			log.Logger.Info("Conectando ao relay", zap.String("url", url))
			client, err := nostr.RelayConnect(ctx, url)
			if err != nil {
				log.Logger.Error("Erro ao conectar ao relay", zap.Error(err), zap.String("url", url))
				wg.Done()
			}
			defer client.Close()

			until := nostr.Now()
			fetchAndStoreEvents(ctx, client, cf.PublicKey, &until, cf.Mentioned, cf.Kinds, cf.Tags)
			log.Logger.Debug("Download concluído", zap.String("url", url), zap.String("publicKey", cf.PublicKey))
			wg.Done()
		}()
	}

	wg.Wait()
}

func normalizePublicKey(pk string) (string, error) {
	if strings.HasPrefix(pk, "npub") {
		_, raw, err := nip19.Decode(pk)
		if err != nil {
			return "", err
		}
		return raw.(string), nil
	}
	return pk, nil
}

func fetchAndStoreEvents(ctx context.Context, client *nostr.Relay, pubKey string, until *nostr.Timestamp, mentioned bool, kinds []int, tags []string) {
	nUntil := until.Time().Unix()
	eCount := 0
	for {
		// Define novo filtro para paginação
		filter := nostr.Filter{
			Until: until,
			Limit: pageSize,
			Kinds: kinds,
		}
		if pubKey != "" && !mentioned {
			filter.Authors = []string{pubKey}
		}
		if mentioned {
			filter.Tags = nostr.TagMap{
				"p": []string{pubKey},
			}
		}
		if len(tags) > 0 {
			for _, tag := range tags {
				filter.Tags = nostr.TagMap{
					"t": []string{tag},
				}
			}
		}

		pageCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		sub, err := client.Subscribe(pageCtx, []nostr.Filter{filter})
		if err != nil {
			log.Logger.Error("Erro ao assinar o filtro", zap.Error(err), zap.String("url", client.URL))
			cancel()
			return
		}

		var count int
		var lastTimestamp int64 = nUntil
		for evt := range sub.Events {
			if evt.CreatedAt.Time().Unix() < lastTimestamp {
				lastTimestamp = evt.CreatedAt.Time().Unix()
			}
			count++
			if err := db.DbQueries.InsertEvent(ctx, evt); err != nil {
				log.Logger.Error("Erro ao salvar evento", zap.Error(err), zap.String("id", evt.ID))
			}
		}

		eCount += count

		if count < pageSize {
			log.Logger.Info(
				"Nenhum evento encontrado ou fim da paginação",
				zap.String("url", client.URL),
				zap.Int("total", eCount),
			)
			break
		}

		nUntil = lastTimestamp - 1 // paginação manual (busca eventos mais antigos)

		log.Logger.Info(
			"Paginação: buscando próxima página",
			zap.Int("eventos", count),
			zap.Int64("until", nUntil),
			zap.String("url", client.URL),
			zap.Int("total", eCount),
		)
	}
}
