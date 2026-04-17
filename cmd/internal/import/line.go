package _import

import (
	"bufio"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

func lineWorker(cf *ConfImport, jobs <-chan Job, errors chan<- ErrorInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		var data nostr.Event
		if err := unmarshalEvent([]byte(job.Line), &data); err != nil {
			errors <- ErrorInfo{LineNumber: job.LineNumber, Err: fmt.Errorf("decode: %w", err)}
			continue
		}
		if err := saveToDatabase(cf.ctx, cf.dbc, &data); err != nil {
			errors <- ErrorInfo{LineNumber: job.LineNumber, Err: fmt.Errorf("persistência: %w", err)}
		}
	}
}

func processLineByLine(cf *ConfImport) (int, error) {
	fmt.Println("Modo: linha a linha (paralelo)")
	if cf.numWorkers <= 0 {
		cf.numWorkers = runtime.NumCPU()
	}
	jobs := make(chan Job, cf.numWorkers*2)
	errors := make(chan ErrorInfo, 100)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup

	for i := 0; i < cf.numWorkers; i++ {
		wg.Add(1)
		go lineWorker(cf, jobs, errors, &wg)
	}

	go func() {
		file, err := os.Open(cf.filename)
		if err != nil {
			errors <- ErrorInfo{LineNumber: 0, Err: fmt.Errorf("abrir arquivo: %w", err)}
			close(jobs)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNumber := 1

	READ:
		for scanner.Scan() {
			select {
			case <-stopChan:
				fmt.Println("\nInterrupção recebida. Encerrando leitura.")
				break READ
			default:
				jobs <- Job{LineNumber: lineNumber, Line: scanner.Text()}
				lineNumber++
			}
		}
		close(jobs)

		if err := scanner.Err(); err != nil {
			errors <- ErrorInfo{LineNumber: 0, Err: fmt.Errorf("ler arquivo: %w", err)}
		}
	}()

	go func() {
		wg.Wait()
		close(errors)
	}()

	count := reportErrors(errors)
	if count > 0 && cf.failOnErr {
		return count, fmt.Errorf("line import finished with %d errors", count)
	}

	return count, nil
}
