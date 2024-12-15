package cmd

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/spf13/cobra"
	"log"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with initial data",
	Run:   runSeed,
}

func runSeed(cmd *cobra.Command, args []string) {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Erro ao carregar a configuração: %v", err)
	}

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	// Iniciar Conexão com o banco de dados
	if err := db.Init(mainCtx); err != nil {
		log.Fatalf("Erro ao iniciar conexão com o banco de dados: %v", err)
	}

	if err := db.DbQueries.Migrate(mainCtx); err != nil {
		log.Fatalf("Erro ao migrar o banco de dados: %v", err)
	} else {
		log.Println("Banco de dados migrado com sucesso")
	}
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
