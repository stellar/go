package cmd

import (
	"testing"

	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDBCommandsTestSuite(t *testing.T) {
	dbCmdSuite := &DBCommandsTestSuite{}
	suite.Run(t, dbCmdSuite)
}

type DBCommandsTestSuite struct {
	suite.Suite
	dsn string
}

func (s *DBCommandsTestSuite) SetupSuite() {
	runDBReingestRangeFn = func([]history.LedgerRange, bool, uint,
		horizon.Config, ingest.StorageBackendConfig) error {
		return nil
	}

	newDB := dbtest.Postgres(s.T())
	s.dsn = newDB.DSN

	RootCmd.SetArgs([]string{
		"db", "migrate", "up", "--db-url", s.dsn})
	require.NoError(s.T(), RootCmd.Execute())
}

func (s *DBCommandsTestSuite) TestDefaultParallelJobSizeForBufferedBackend() {
	RootCmd.SetArgs([]string{
		"db", "reingest", "range",
		"--db-url", s.dsn,
		"--network", "testnet",
		"--parallel-workers", "2",
		"--ledgerbackend", "datastore",
		"--datastore-config", "../config.storagebackend.toml",
		"2",
		"10"})

	require.NoError(s.T(), dbReingestRangeCmd.Execute())
	require.Equal(s.T(), parallelJobSize, uint32(100))
}

func (s *DBCommandsTestSuite) TestDefaultParallelJobSizeForCaptiveBackend() {
	RootCmd.SetArgs([]string{
		"db", "reingest", "range",
		"--db-url", s.dsn,
		"--network", "testnet",
		"--stellar-core-binary-path", "/test/core/bin/path",
		"--parallel-workers", "2",
		"--ledgerbackend", "captive-core",
		"2",
		"10"})

	require.NoError(s.T(), RootCmd.Execute())
	require.Equal(s.T(), parallelJobSize, uint32(100_000))
}

func (s *DBCommandsTestSuite) TestUsesParallelJobSizeWhenSetForCaptive() {
	RootCmd.SetArgs([]string{
		"db", "reingest", "range",
		"--db-url", s.dsn,
		"--network", "testnet",
		"--stellar-core-binary-path", "/test/core/bin/path",
		"--parallel-workers", "2",
		"--parallel-job-size", "5",
		"--ledgerbackend", "captive-core",
		"2",
		"10"})

	require.NoError(s.T(), RootCmd.Execute())
	require.Equal(s.T(), parallelJobSize, uint32(5))
}

func (s *DBCommandsTestSuite) TestUsesParallelJobSizeWhenSetForBuffered() {
	RootCmd.SetArgs([]string{
		"db", "reingest", "range",
		"--db-url", s.dsn,
		"--network", "testnet",
		"--parallel-workers", "2",
		"--parallel-job-size", "5",
		"--ledgerbackend", "datastore",
		"--datastore-config", "../config.storagebackend.toml",
		"2",
		"10"})

	require.NoError(s.T(), RootCmd.Execute())
	require.Equal(s.T(), parallelJobSize, uint32(5))
}
