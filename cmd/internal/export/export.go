package export

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Options struct {
	Filename  string
	BatchSize int
}

func Export(opt *Options) error {
	if err := loadAndInit(); err != nil {
		return err
	}

	ctx, cancel := setupContext()
	defer cancel()

	point, err := streamAndWriteEvents(ctx, opt)
	if err != nil {
		return err
	}

	log.Logger.Info("Exportação concluída com sucesso", zap.Int64("total_exported", point))
	return nil
}

func loadAndInit() error {
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Erro ao carregar a configuração: %v", err)
		return err
	}

	log.Init()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Init(ctx); err != nil {
		log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
		return err
	}

	return nil
}

func setupContext() (context.Context, context.CancelFunc) {
	mainCtx := context.Background()
	ctx, cancel := context.WithCancel(mainCtx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Logger.Info("Recebido sinal de parada, finalizando exportação...")
		cancel()
	}()

	return ctx, func() {
		cancel()
	}
}

func streamAndWriteEvents(ctx context.Context, opt *Options) (int64, error) {

	writer := &bytes.Buffer{}
	w := NewWriter(writer)

	var count int64
	evChan := db.DbQueries.StreamAllEvents(ctx, opt.BatchSize)

	for batch := range evChan {
		select {
		case <-ctx.Done():
			log.Logger.Info("Exportação cancelada pelo usuário")
			return count, ctx.Err()
		default:
			for _, ev := range *batch {
				if err := w.Write(ev); err != nil {
					log.Logger.Error("Erro ao escrever evento no arquivo", zap.Error(err))
					return count, fmt.Errorf("failed to write event to file: %w", err)
				}
				count++
			}
			err := writeBytesToFileAppend(opt.Filename, writer)
			if err != nil {
				log.Logger.Error("Erro ao escrever arquivo de exportação", zap.Error(err))
				return count, fmt.Errorf("failed to write export file: %w", err)
			} else {
				writer.Reset() // limpa o buffer após cada lote
			}
		}
	}

	return count, nil
}
func writeBytesToFileAppend(filename string, w *bytes.Buffer) error {
	// Abrir (ou criar) o arquivo só para escrita, posicionando o cursor no fim
	f, err := os.OpenFile(
		filename,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o644, // permissões: rw-r--r--
	)
	if err != nil {
		return fmt.Errorf("abrindo %s: %w", filename, err)
	}
	defer f.Close()

	// Usamos bufio.Writer para evitar syscalls excessivas em lotes grandes
	bw := bufio.NewWriter(f)
	if _, err := bw.Write(w.Bytes()); err != nil {
		return fmt.Errorf("gravando em %s: %w", filename, err)
	}
	if err := bw.Flush(); err != nil { // garante escrita em disco
		return fmt.Errorf("flush em %s: %w", filename, err)
	}

	return nil
}
