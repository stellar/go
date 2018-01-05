package main

import (
	"net/http"
	"os"
	"time"

	"github.com/facebookgo/inject"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/bridge/database"
	"github.com/stellar/go/services/bridge/listener"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// Config contains config params of the bridge server
type Config struct {
	Horizon  string `valid:"required"`
	Database struct {
		Type string `valid:"required"`
		DSN  string `valid:"required"`
	} `valid:"required"`
	Accounts struct {
		ReceivingAccount string `valid:"required" toml:"receiving_account"`
	} `valid:"required"`
	Callbacks struct {
		Receive string `valid:"required"`
	} `valid:"required"`
}

func readConfig(cfgPath string) Config {
	var cfg Config

	err := config.Read(cfgPath, &cfg)
	if err != nil {
		switch cause := errors.Cause(err).(type) {
		case *config.InvalidConfigError:
			log.Error("config file: ", cause)
		default:
			log.Error(err)
		}
		os.Exit(-1)
	}

	return cfg
}

func main() {
	cfg := readConfig("bridge.toml")

	var g inject.Graph

	db := &database.PostgresDatabase{}
	err := db.Open(cfg.Database.DSN)
	if err != nil {
		log.WithField("err", err).Error("Error connecting to database")
		os.Exit(-1)
	}

	httpClientWithTimeout := http.Client{
		Timeout: 30 * time.Second,
	}

	horizonClient := &horizon.Client{
		URL:  cfg.Horizon,
		HTTP: &httpClientWithTimeout,
	}

	listener := &listener.PaymentListener{
		ReceivingAccount: cfg.Accounts.ReceivingAccount,
	}

	err = g.Provide(
		&inject.Object{Value: &cfg},
		&inject.Object{Value: db},
		&inject.Object{Value: horizonClient},
		&inject.Object{Value: &httpClientWithTimeout},
		&inject.Object{Value: listener},
	)
	if err != nil {
		log.WithField("err", err).Error("Error providing objects to injector")
		os.Exit(-1)
	}

	if err := g.Populate(); err != nil {
		log.WithField("err", err).Error("Error injecting objects")
		os.Exit(-1)
	}

	err = listener.Listen()
	if err != nil {
		log.WithField("err", err).Error("Error starting payment listener")
		os.Exit(-1)
	}
}
