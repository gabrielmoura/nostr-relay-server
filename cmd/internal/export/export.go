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
	"sync"
	"syscall"
	"time"
)

type Options struct {
	Filename      string
	BatchSize     int
	WriterWorkers int
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

	return ctx, cancel
}

func streamAndWriteEvents(ctx context.Context, opt *Options) (int64, error) {
	writeJobs := make(chan *bytes.Buffer, opt.WriterWorkers*2)
	var wg sync.WaitGroup
	var count int64
	var countMu sync.Mutex

	// Buffer pool
	bufferPool := sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}

	// Start writer workers
	for i := 0; i < opt.WriterWorkers; i++ {
		wg.Add(1)
		go func(id int, pool *sync.Pool) {
			defer wg.Done()
			for buf := range writeJobs {
				if err := writeBytesToFileAppend(opt.Filename, buf); err != nil {
					log.Logger.Error("Erro ao escrever arquivo de exportação", zap.Error(err))
				}
				pool.Put(buf) // Reutiliza o buffer
			}
		}(i, &bufferPool)
	}

	evChan := db.DbQueries.StreamAllEvents(ctx, opt.BatchSize)

	for batch := range evChan {
		select {
		case <-ctx.Done():
			log.Logger.Info("Exportação cancelada pelo usuário")
			close(writeJobs)
			wg.Wait()
			return count, ctx.Err()
		default:
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()

			w := NewWriter(buf)
			for _, ev := range *batch {
				if err := w.Write(ev); err != nil {
					log.Logger.Error("Erro ao escrever evento no buffer", zap.Error(err))
					close(writeJobs)
					wg.Wait()
					return count, fmt.Errorf("failed to write event to buffer: %w", err)
				}
			}

			writeJobs <- buf

			countMu.Lock()
			count += int64(len(*batch))
			countMu.Unlock()
		}
	}

	close(writeJobs)
	wg.Wait()

	return count, nil
}

func writeBytesToFileAppend(filename string, buf *bytes.Buffer) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("abrindo %s: %w", filename, err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	if _, err := bw.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("gravando em %s: %w", filename, err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush em %s: %w", filename, err)
	}
	return nil
}
