package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	cronjob "github.com/gabrielmoura/nostr-relay-server/infra/cron"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Run cron jobs",
	Run:   runCron,
}

func init() {
	rootCmd.AddCommand(cronCmd)
}

func runCron(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		fmt.Println("Erro ao carregar a configuração:", err)
		return
	}

	log.Init()

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := db.Init(mainCtx); err != nil {
		log.Logger.Fatal("erro ao iniciar conexão com o banco de dados", zap.Error(err))
	}

	c := cron.New(cron.WithSeconds())

	if !config.Cfg.Cron.Enabled {
		log.Logger.Info("cron desabilitado por configuração", zap.Bool("cron.enabled", false))
		return
	}

	registerCronJobs(c, mainCtx)

	if len(c.Entries()) == 0 {
		log.Logger.Warn("nenhum job de cron foi registrado; verifique configuração")
		return
	}

	c.Start()
	log.Logger.Info("cron scheduler iniciado", zap.Int("jobs", len(c.Entries())))

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Logger.Info("sinal de desligamento recebido para cron scheduler")
	cancel()
	stopCtx := c.Stop()
	<-stopCtx.Done()
	log.Logger.Info("cron scheduler finalizado")
}

func registerCronJobs(c *cron.Cron, baseCtx context.Context) {
	register := func(name string, schedule string, job func(context.Context) error) {
		if schedule == "" {
			log.Logger.Warn("cron job sem schedule, ignorado", zap.String("job", name))
			return
		}

		_, err := c.AddFunc(schedule, func() {
			jobCtx, cancel := context.WithTimeout(baseCtx, 30*time.Minute)
			defer cancel()

			start := time.Now()
			if err := job(jobCtx); err != nil {
				log.Logger.Error("cron job failed", zap.String("job", name), zap.Error(err), zap.Duration("duration", time.Since(start)))
				return
			}
			log.Logger.Info("cron job finished", zap.String("job", name), zap.Duration("duration", time.Since(start)))
		})
		if err != nil {
			log.Logger.Error("failed to register cron job", zap.String("job", name), zap.String("schedule", schedule), zap.Error(err))
			return
		}

		log.Logger.Info("cron job registered", zap.String("job", name), zap.String("schedule", schedule))
	}

	if config.Cfg.Cron.DBOptimization.Enabled {
		register("db_optimization", config.Cfg.Cron.DBOptimization.Schedule, cronjob.RunDBOptimization)
	}

	if config.Cfg.Cron.ReportedEventsFetch.Enabled {
		register("reported_events_fetch", config.Cfg.Cron.ReportedEventsFetch.Schedule, cronjob.FetchReportedEvents)
	}

	if config.Cfg.Cron.DeleteOldEvents.Enabled {
		register("delete_old_events", config.Cfg.Cron.DeleteOldEvents.Schedule, cronjob.DeleteOldEvent)
	}
}
