package main

import (
	"context"
	"fmt"
	htttpp "net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/spf13/cobra"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"

	"github.com/stellar/go/handlers/txsub"
)

// Config represents the configuration of a transctions submission service
type Config struct {
	Port              int    `valid:"required"`
	Horizonurl        string `valid:"required"`
	Networkpassphrase string `valid:"required"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "txsub",
		Short: "stellar transaction submission service",
		Long: `
The Stellar transaction submission service let's you easilly submit transactions
to a configurable horizon server or a configurable stellar core client.
    `,
		Run: run,
	}

	rootCmd.PersistentFlags().String("conf", "./txsub.cfg", "config file path")
	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	var (
		cfg     Config
		cfgPath = cmd.PersistentFlags().Lookup("conf").Value.String()
	)
	log.SetLevel(log.InfoLevel)
	err := config.Read(cfgPath, &cfg)

	if err != nil {
		switch cause := errors.Cause(err).(type) {
		case *config.InvalidConfigError:
			log.Error("config file: ", cause)
		default:
			log.Error(err)
		}
		os.Exit(1)
	}

	driver := initDriver(cfg)

	mux := initMux(driver)
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)

	http.Run(http.Config{
		ListenAddr: addr,
		Handler:    mux,
		OnStarting: func() {
			log.Infof("starting txsub service - %s", app.Version())
			log.Infof("listening on %s", addr)
		},
	})
}

func initDriver(cfg Config) txsub.Driver {
	client := horizon.Client{
		URL:  cfg.Horizonurl,
		HTTP: htttpp.DefaultClient,
	}

	return txsub.InitHorizonProxyDriver(client, cfg.Networkpassphrase)
}

func initMux(driver txsub.Driver) *chi.Mux {
	mux := http.NewAPIMux(false)

	t := txsub.Handler{
		Driver:  driver,
		Ticks:   time.NewTicker(1 * time.Second),
		Context: context.Background(),
	}

	mux.Post("/tx", t.ServeHTTP)
	mux.Post("/tx/", t.ServeHTTP)

	go t.Run()

	return mux
}
