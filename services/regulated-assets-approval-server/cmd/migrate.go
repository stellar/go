package cmd

import (
	"go/types"
	"strconv"
	"strings"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbmigrate"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/log"
)

type MigrateCommand struct {
	DatabaseURL string
}

func (c *MigrateCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "database-url",
			Usage:       "Database URL",
			OptType:     types.String,
			ConfigKey:   &c.DatabaseURL,
			FlagDefault: "postgres://localhost:5432/?sslmode=disable",
			Required:    true,
		},
	}
	cmd := &cobra.Command{
		Use:   "migrate [up|down] [count]",
		Short: "Run migrations on the database",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c.Migrate(cmd, args)
		},
	}
	configOpts.Init(cmd)

	return cmd
}

func (c *MigrateCommand) Migrate(cmd *cobra.Command, args []string) {
	db, err := db.Open(c.DatabaseURL)
	if err != nil {
		log.Errorf("Error opening database: %s", err.Error())
		return
	}

	if len(args) < 1 {
		cmd.Help()
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
		log.Errorf("Invalid migration direction, must be 'up' or 'down'.")
		return
	}

	var count int
	if len(args) >= 2 {
		count, err = strconv.Atoi(args[1])
		if err != nil {
			log.Errorf("Invalid migration count, must be a number.")
			return
		}
		if count < 1 {
			log.Errorf("Invalid migration count, must be a number greater than zero.")
			return
		}
	}

	migrations, err := dbmigrate.PlanMigration(db, dir, count)
	if err != nil {
		log.Errorf("Error planning migration: %s", err.Error())
		return
	}
	if len(migrations) > 0 {
		log.Infof("Migrations to apply %s: %s", dirStr, strings.Join(migrations, ", "))
	}

	n, err := dbmigrate.Migrate(db, dir, count)
	if err != nil {
		log.Errorf("Error applying migrations: %s", err.Error())
		return
	}
	if n > 0 {
		log.Infof("Successfully applied %d migrations %s.", n, dirStr)
	} else {
		log.Infof("No migrations applied %s.", dirStr)
	}
}
