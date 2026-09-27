package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/repository"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/search/meilisearch"
	"github.com/spf13/cobra"
)

var searchReindexCmd = &cobra.Command{
	Use:   "search-reindex",
	Short: "rebuild the public article Meilisearch index",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		engine := ormInit.GetEngine()
		if engine == nil {
			return errors.New("database engine could not be initialized")
		}
		defer engine.Close()
		indexer := search.NewMeiliSearcher(new(config.MeiliSearch).MeiliSearch())
		maintenance, err := service.NewArticleSearchService(repository.NewArticleRepo(engine), indexer)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
		defer cancel()
		if err := maintenance.Reconcile(ctx); err != nil {
			return fmt.Errorf("reconcile article search index: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "public article search index is reconciled")
		return err
	},
}

func init() {
	rootCmd.AddCommand(searchReindexCmd)
}
