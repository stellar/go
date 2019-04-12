package main

import (
	"context"
	"errors"
	"os"

	_ "github.com/lib/pq"
	horizonclient "github.com/stellar/go/exp/clients/horizon"
	ticker "github.com/stellar/go/exp/ticker/internal"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
	"github.com/stellar/go/exp/ticker/internal/utils"
)

func main() {
	dbInfo := os.Getenv("DB_INFO")
	if dbInfo == "" {
		panic(errors.New("could not start: env var DB_INFO not provided"))
	}
	session, err := tickerdb.CreateSession("postgres", dbInfo)
	defer session.DB.Close()
	utils.PanicIfError(err)

	client := horizonclient.DefaultPublicNetClient

	err = ticker.RefreshAssets(&session, client)
	utils.PanicIfError(err)

	err = ticker.BackfillTrades(&session, client, 2, 0)
	utils.PanicIfError(err)

	ctx := context.Background()
	err = ticker.StreamTrades(ctx, &session, client)
}
