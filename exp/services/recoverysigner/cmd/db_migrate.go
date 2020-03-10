package cmd

import (
	"go/types"
	"strconv"
	"strings"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
	dbpkg "github.com/stellar/go/exp/services/recoverysigner/internal/db"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbmigrate"
	"github.com/stellar/go/support/config"
	supportlog "github.com/stellar/go/support/log"
)

type DBMigrateCommand struct {
	Logger *supportlog.Entry
}

type Options struct {
	Logger      *supportlog.Entry
	DatabaseURL string
}

func (c *DBMigrateCommand) Command() *cobra.Command {
	opts := Options{
		Logger: c.Logger,
	}
	configOpts := config.ConfigOptions{
		{
			Name:        "db-url",
			Usage:       "Database URL",
			OptType:     types.String,
			ConfigKey:   &opts.DatabaseURL,
			FlagDefault: "postgres://localhost:5432/?sslmode=disable",
			Required:    true,
		},
	}
	cmd := &cobra.Command{
		Use:   "migrate [up|down] [count]",
		Short: "Run migrations on the database",
		Run: func(_ *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Run(opts, args)
		},
	}
	configOpts.Init(cmd)
	return cmd
}

func (c *DBMigrateCommand) Run(opts Options, args []string) {
	db, err := dbpkg.Open(opts.DatabaseURL)
	if err != nil {
		c.Logger.Errorf("Error opening database: %s", err.Error())
		return
	}

	if len(args) < 1 {
		c.Logger.Errorf("No migration direction, must be 'up' or 'down'.")
		return
	}
	dirStr := args[0]

	var dir migrate.MigrationDirection
	switch dirStr {
	case "down":
		dir = migrate.Down
	case "up":
		dir = migrate.Up
	default:
		c.Logger.Errorf("Invalid migration direction, must be 'up' or 'down'.")
		return
	}

	var count int
	if len(args) >= 2 {
		count, err = strconv.Atoi(args[1])
		if err != nil {
			c.Logger.Errorf("Invalid migration count, must be a number or not provided.")
			return
		}
	}

	migrations, err := dbmigrate.PlanMigration(db, dir, count)
	if err != nil {
		c.Logger.Errorf("Error planning migration: %s", err.Error())
		return
	}
	if len(migrations) > 0 {
		c.Logger.Infof("Migrations to apply %s: %s", dirStr, strings.Join(migrations, ", "))
	}

	n, err := dbmigrate.Migrate(db, dir, count)
	if err != nil {
		c.Logger.Errorf("Error applying migrations: %s", err.Error())
		return
	}
	if n > 0 {
		c.Logger.Infof("Successfully applied %d migrations %s.", n, dirStr)
	} else {
		c.Logger.Infof("No migrations applied %s.", dirStr)
	}
}
