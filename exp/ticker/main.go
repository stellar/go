package main

import (
	"errors"
	"os"

	_ "github.com/lib/pq"
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

	err = ticker.RefreshAssets(&session)
	utils.PanicIfError(err)
}
