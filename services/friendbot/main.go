package main

import (
	"database/sql"
	"fmt"
	stdhttp "net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/spf13/cobra"
	"github.com/stellar/go/services/friendbot/internal"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/env"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
)

// Config represents the configuration of a friendbot server
type Config struct {
	Port                   int         `toml:"port" valid:"required"`
	FriendbotSecret        string      `toml:"friendbot_secret" valid:"required,stellar_seed"`
	NetworkPassphrase      string      `toml:"network_passphrase" valid:"required"`
	HorizonURL             string      `toml:"horizon_url" valid:"required"`
	StartingBalance        string      `toml:"starting_balance" valid:"required,stellar_amount"`
	TLS                    *config.TLS `valid:"optional"`
	NumMinions             int         `toml:"num_minions" valid:"optional"`
	BaseFee                int64       `toml:"base_fee" valid:"optional"`
	MinionBatchSize        int         `toml:"minion_batch_size" valid:"optional"`
	SubmitTxRetriesAllowed int         `toml:"submit_tx_retries_allowed" valid:"optional"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "friendbot",
		Short: "friendbot for the Stellar Test Network",
		Long:  "Client-facing API server for the friendbot service on the Stellar Test Network",
		Run:   run,
	}

	rootCmd.PersistentFlags().String("conf", "", "config file path")
	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	var (
		cfg     Config
		cfgPath = cmd.PersistentFlags().Lookup("conf").Value.String()
	)
	log.SetLevel(log.InfoLevel)

	// Set config values based on env vars with defaults where possible
	cfg.Port = env.Int("PORT", 8000)
	cfg.FriendbotSecret = env.String("FRIENDBOT_SECRET", "")
	cfg.NetworkPassphrase = env.String("NETWORK_PASSPHRASE", "")
	cfg.HorizonURL = env.String("HORIZON_URL", "")
	cfg.StartingBalance = env.String("STARTING_BALANCE", "10000")
	cfg.NumMinions = env.Int("NUM_MINIONS", 0)
	cfg.BaseFee = int64(env.Int("BASE_FEE", 100000))
	cfg.MinionBatchSize = env.Int("MINION_BATCH_SIZE", 0)
	cfg.SubmitTxRetriesAllowed = env.Int("SUBMIT_TX_RETRIES_ALLOWED", 0)

	var err error
	if cfgPath != "" {
		err = config.Read(cfgPath, &cfg)
	} else {
		err = config.Validate(&cfg)
	}
	if err != nil {
		switch cause := errors.Cause(err).(type) {
		case *config.InvalidConfigError:
			log.Error("config file: ", cause)
		default:
			log.Error(err)
		}
		os.Exit(1)
	}

	fb, err := initFriendbot(cfg.FriendbotSecret, cfg.NetworkPassphrase, cfg.HorizonURL, cfg.StartingBalance,
		cfg.NumMinions, cfg.BaseFee, cfg.MinionBatchSize, cfg.SubmitTxRetriesAllowed)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	router := initRouter(fb)
	registerProblems()

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)

	http.Run(http.Config{
		ListenAddr: addr,
		Handler:    router,
		TLS:        cfg.TLS,
		OnStarting: func() {
			log.Infof("starting friendbot server - %s", app.Version())
			log.Infof("listening on %s", addr)
		},
	})
}

func initRouter(fb *internal.Bot) *chi.Mux {
	mux := http.NewAPIMux(log.DefaultLogger)

	handler := &internal.FriendbotHandler{Friendbot: fb}
	mux.Get("/", handler.Handle)
	mux.Post("/", handler.Handle)
	mux.NotFound(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		problem.Render(r.Context(), w, problem.NotFound)
	}))

	return mux
}

func registerProblems() {
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)

	accountExistsProblem := problem.BadRequest
	accountExistsProblem.Detail = internal.ErrAccountExists.Error()
	problem.RegisterError(internal.ErrAccountExists, accountExistsProblem)
}
