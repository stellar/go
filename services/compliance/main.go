package main

import (
	"fmt"

	"goji.io"
	"goji.io/pat"

	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/stellar/go/handlers/compliance"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
)

// YesStrategy sends OK status to every valid request.
// It's for testing purposes only and should not be merged
// to master.
type YesStrategy struct{}

// SanctionsCheck ...
func (s *YesStrategy) SanctionsCheck(data compliance.AuthData, response *compliance.AuthResponse) error {
	response.TxStatus = compliance.AuthStatusOk
	return nil
}

// GetUserData ...
func (s *YesStrategy) GetUserData(data compliance.AuthData, response *compliance.AuthResponse) error {
	response.InfoStatus = compliance.AuthStatusOk
	response.DestInfo = `{"name": "John Doe"}`
	return nil
}

// PersistTransaction ...
func (s *YesStrategy) PersistTransaction(data compliance.AuthData) error {
	return nil
}

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

	mux := initMux(&YesStrategy{})
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

	authHandler := &compliance.AuthHandler{strategy}

	mux.Handle(pat.Post("/auth"), authHandler)
	mux.Handle(pat.Post("/auth/"), authHandler)

	return mux
}
