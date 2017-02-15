package main

import (
	"fmt"

	"goji.io"
	"goji.io/pat"

	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/stellar/go/handlers/compliance"
	complianceProtocol "github.com/stellar/go/protocols/compliance"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "compliance",
		Short: "stellar compliance server",
		Long:  `TODO`,
		Run:   run,
	}

	rootCmd.PersistentFlags().String("conf", "./federation.cfg", "config file path")
	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	log.SetLevel(log.InfoLevel)

	strategy := &compliance.CallbackStrategy{
		SanctionsCheckURL: "abc",
		GetUserDataURL:    "def",
	}

	mux := initMux(strategy)
	addr := fmt.Sprintf("0.0.0.0:%d", 8000)

	http.Run(http.Config{
		ListenAddr: addr,
		Handler:    mux,
		OnStarting: func() {
			log.Infof("starting compliance server - %s", app.Version())
			log.Infof("listening on %s", addr)
		},
	})
}

func initMux(strategy compliance.Strategy) *goji.Mux {
	mux := goji.NewMux()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"*"},
	})
	mux.Use(c.Handler)
	mux.Use(log.HTTPMiddleware)

	authHandler := &compliance.AuthHandler{
		Strategy: strategy,
		PersistTransaction: func(data complianceProtocol.AuthData) error {
			fmt.Println("Persist")
			return nil
		},
	}

	mux.Handle(pat.Post("/auth"), authHandler)
	mux.Handle(pat.Post("/auth/"), authHandler)

	return mux
}
