package main

import (
	"context"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/services/submitter/internal"
	"github.com/stellar/go/support/db"
)

func main() {
	sqlxDB, err := sqlx.Connect("postgres", "dbname=submitter sslmode=disable")
	if err != nil {
		return
	}
	store := internal.PostgresStore{
		Session: &db.Session{
			DB: sqlxDB,
		},
	}
	numChannels, err := strconv.ParseUint(os.Getenv("SUBMITTER_NUM_CHANNELS"), 10, 64)
	if err != nil || numChannels > 1000 {
		return
	}
	baseFee, err := strconv.ParseUint(os.Getenv("SUBMITTER_MAX_BASE_FEE"), 10, 64)
	if err != nil {
		return
	}
	ts := internal.TransactionSubmitter{
		Horizon:         horizonclient.DefaultTestNetClient,
		Network:         network.TestNetworkPassphrase,
		NumChannels:     uint(numChannels),
		Store:           store,
		RootAccountSeed: os.Getenv("SUBMITTER_ROOT_SEED"),
		MaxBaseFee:      uint(baseFee),
	}
	ts.Start(context.Background())
}
