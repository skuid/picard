package cmd

import (
	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
	"github.com/skuid/picard"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations up or down",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Run migrations upwards from current version",
	Run:   up,
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Run migrations downwards from current version",
	Run:   down,
}

func init() {
	RootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
}

func getMigrator() *migrate.Migrate {
	db := picard.GetConnection()
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		zap.L().Fatal("MIGRATION ERROR: Failed to initialize pg driver!", zap.Error(err))
	}
	migrator, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		zap.L().Fatal("MIGRATION ERROR: Failed to initialize migration tool!", zap.Error(err))
	}
	return migrator
}

func migrateUp(migrator *migrate.Migrate) error {
	err := migrator.Up()
	errDirty, isDirty := err.(migrate.ErrDirty)
	if isDirty {
		zap.L().Info("Migrations are in a dirty state, forcing to previous version and re-running")
		// Force to the version prior to the dirty version and re-run.
		migrator.Force(errDirty.Version - 1)
		return migrateUp(migrator)
	}
	return err
}

func up(cmd *cobra.Command, args []string) {
	zap.L().Info("Attempting upwards migrations")
	migrator := getMigrator()
	if err := migrateUp(migrator); err != nil {
		switch err {
		case migrate.ErrNoChange:
			zap.L().Info("No migrations to run, database is current")
		default:
			zap.L().Fatal("MIGRATION ERROR: Failed while running migrations.", zap.Error(err))
		}
	}
	zap.L().Info("Finished upwards migrations")
}

func down(cmd *cobra.Command, args []string) {
	zap.L().Info("Attempting downwards migrations")
	migrator := getMigrator()
	if err := migrator.Down(); err != nil {
		zap.L().Fatal("MIGRATION ERROR: Failed while running migrations.", zap.Error(err))
	}
	zap.L().Info("Finished downwards migrations")
}
