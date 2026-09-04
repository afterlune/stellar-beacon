package cmd

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/persistence/migration"
	"benetnasch/app/infra/persistence/ormInit"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "apply pending database migrations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		allowWrites, err := cmd.Flags().GetBool("allow-writes")
		if err != nil {
			return err
		}
		if err := requireMigrationWriteAuthorization(allowWrites); err != nil {
			return err
		}
		return runMigrations(cmd.Context())
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "inspect migration state without modifying the database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigrationStatus(cmd.Context(), cmd.OutOrStdout())
	},
}

func init() {
	migrateCmd.Flags().Bool("allow-writes", false, "authorize applying database migrations")
	migrateCmd.AddCommand(migrateStatusCmd)
	rootCmd.AddCommand(migrateCmd)
}

func requireMigrationWriteAuthorization(allowWrites bool) error {
	if !allowWrites {
		return fmt.Errorf("migrate requires explicit --allow-writes")
	}
	return nil
}

func runMigrations(ctx context.Context) error {
	if err := config.ValidateDatabase(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	engine := ormInit.GetEngine()
	if engine == nil {
		return errors.Unavailable("migration.database", nil)
	}
	if err := migration.NewRunner(engine).Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func runMigrationStatus(ctx context.Context, output io.Writer) error {
	if err := config.ValidateDatabase(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	engine := ormInit.GetEngine()
	if engine == nil {
		return errors.Unavailable("migration.database", nil)
	}
	plan, err := migration.NewRunner(engine).Status(ctx)
	if err != nil {
		return fmt.Errorf("inspect migrations: %w", err)
	}
	if output == nil {
		return fmt.Errorf("migration status output is nil")
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(plan); err != nil {
		return fmt.Errorf("encode migration status: %w", err)
	}
	return nil
}
