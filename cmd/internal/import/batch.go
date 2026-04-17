package _import

import (
	"bufio"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

func batchWorker(cf *ConfImport, batchChan <-chan Batch, errorChan chan<- ErrorInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for batch := range batchChan {
		if err := saveBatchToDatabase(cf.ctx, cf.dbc, batch.Items); err != nil {
			for _, ln := range batch.LineNumbers {
				errorChan <- ErrorInfo{LineNumber: ln, Err: fmt.Errorf("persistência: %w", err)}
			}
		}
	}
}

func processInBatches(cf *ConfImport) (int, error) {
	fmt.Printf("Modo: batch (tamanho do lote = %d)\n", cf.batchSize)
	if cf.numWorkers <= 0 {
		cf.numWorkers = runtime.NumCPU()
	}
	batchChan := make(chan Batch, cf.numWorkers*2)
	errorChan := make(chan ErrorInfo, 1000)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup

	for i := 0; i < cf.numWorkers; i++ {
		wg.Add(1)
		go batchWorker(cf, batchChan, errorChan, &wg)
	}

	go func() {
		file, err := os.Open(cf.filename)
		if err != nil {
			errorChan <- ErrorInfo{LineNumber: 0, Err: fmt.Errorf("abrir arquivo: %w", err)}
			close(batchChan)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var batch []*nostr.Event
		var lineNumbers []int
		line := 1

	READ:
		for scanner.Scan() {
			select {
			case <-stopChan:
				fmt.Println("\nInterrupção recebida. Encerrando leitura.")
				break READ
			default:
				text := scanner.Text()
				var item nostr.Event

				if err := unmarshalEvent([]byte(strings.TrimSpace(text)), &item); err != nil {
					errorChan <- ErrorInfo{LineNumber: line, Err: fmt.Errorf("decode: %w", err)}
				} else {
					batch = append(batch, &item)
					lineNumbers = append(lineNumbers, line)

				}

				if len(batch) >= cf.batchSize {
					batchChan <- Batch{Items: batch, LineNumbers: lineNumbers}
					batch = nil
					lineNumbers = nil
				}
				line++
			}
		}

		if len(batch) > 0 {
			batchChan <- Batch{Items: batch, LineNumbers: lineNumbers}
		}
		close(batchChan)
	}()

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	count := reportErrors(errorChan)
	if count > 0 && cf.failOnErr {
		return count, fmt.Errorf("batch import finished with %d errors", count)
	}

	return count, nil
}
