package schema

import (
	"database/sql"
	"errors"
	stdLog "log"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stellar/go/support/db"
)

//go:generate go-bindata -ignore .+\.go$ -pkg schema -o bindata.go ./...

// MigrateDir represents a direction in which to perform schema migrations.
type MigrateDir string

const (
	// MigrateUp causes migrations to be run in the "up" direction.
	MigrateUp MigrateDir = "up"
	// MigrateDown causes migrations to be run in the "down" direction.
	MigrateDown MigrateDir = "down"
	// MigrateRedo causes migrations to be run down, then up
	MigrateRedo MigrateDir = "redo"
)

// Migrations represents all of the schema migration for horizon
var Migrations migrate.MigrationSource = &migrate.AssetMigrationSource{
	Asset:    Asset,
	AssetDir: AssetDir,
	Dir:      "migrations",
}

// Init installs the latest schema into db after clearing it first
func Init(db *db.Session) error {
	return db.ExecAll(string(MustAsset("latest.sql")))
}

// Migrate performs schema migration.  Migrations can occur in one of three
// ways:
//
// - up: migrations are performed from the currently installed version upwards.
// If count is 0, all unapplied migrations will be run.
//
// - down: migrations are performed from the current version downard. If count
// is 0, all applied migrations will be run in a downard direction.
//
// - redo: migrations are first ran downard `count` times, and then are rand
// upward back to the current version at the start of the process. If count is
// 0, a count of 1 will be assumed.
func Migrate(db *sql.DB, dir MigrateDir, count int) (int, error) {
	switch dir {
	case MigrateUp:
		return migrate.ExecMax(db, "postgres", Migrations, migrate.Up, count)
	case MigrateDown:
		return migrate.ExecMax(db, "postgres", Migrations, migrate.Down, count)
	case MigrateRedo:

		if count == 0 {
			count = 1
		}

		down, err := migrate.ExecMax(db, "postgres", Migrations, migrate.Down, count)
		if err != nil {
			return down, err
		}

		return migrate.ExecMax(db, "postgres", Migrations, migrate.Up, down)
	default:
		return 0, errors.New("Invalid migration direction")
	}
}

// GetMigrations, finds the names of any migrations needed in the "up" or "down" directions.
// The differencing step is necessary to handle the "down" case correctly.
func GetMigrations(dbUrl, dirStr string) (result []string) {
	// Migrations can be either to later schema versions (up) or earlier (down)
	directions := map[string]migrate.MigrationDirection{
		"up":   migrate.Up,
		"down": migrate.Down,
	}

	if _, ok := directions[dirStr]; !ok {
		stdLog.Fatalf(`Invalid migration direction "%v": must be "up" or "down"`, dirStr)
	}

	// Get a DB handle
	db, dbErr := sql.Open("postgres", dbUrl)
	if dbErr != nil {
		stdLog.Fatal(dbErr)
	}

	// Get the possible migrations in the given direction
	dir := directions[dirStr]
	possibleMigrations, _, migrateErr := migrate.PlanMigration(db, "postgres", Migrations, dir, 0)
	if migrateErr != nil {
		stdLog.Fatal(migrateErr)
	}

	// Extract a list of the possible migration names
	var possibleIds []string
	for _, m := range possibleMigrations {
		possibleIds = append(possibleIds, m.Id)
	}

	// Get the set of migrations recorded in the database
	migrationRecords, recordErr := migrate.GetMigrationRecords(db, "postgres")
	if recordErr != nil {
		stdLog.Fatal(recordErr)
	}

	// Extract a list of names of the previously applied migrations
	var migrationRecordIds []string
	for _, m := range migrationRecords {
		migrationRecordIds = append(migrationRecordIds, (*m).Id)
	}

	// Find any migrations that need to be applied in this direction
	// migrationsToApply := difference(possibleIds, migrationRecordIds)
	migrationsToApply := migrationRecordIds[len(possibleIds):]

	return migrationsToApply
}

// Return the elements in a that aren't in b
func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}
