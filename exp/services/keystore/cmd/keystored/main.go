package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/stellar/go/exp/services/keystore"
	"github.com/stellar/go/support/log"

	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()

	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "too few arguments")
		os.Exit(1)
	}

	cfg := getConfig()
	// will read the value from AWS parameter store
	if cfg.LogFile != "" {
		logFile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open file to log", err)
			os.Exit(1)
		}

		log.DefaultLogger.Logger.Out = logFile
		log.DefaultLogger.Logger.SetLevel(cfg.LogLevel)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening database", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(cfg.MaxOpenDBConns)
	db.SetMaxIdleConns(cfg.MaxIdleDBConns)

	err = db.Ping()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error accessing database", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "Successfully connected to keystore db")

	cmd := flag.Arg(0)
	switch cmd {
	case "serve":
		_, err := keystore.NewService(ctx, db)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error initializing service object", err)
			os.Exit(1)
		}

		addr := ":8443"
		server := &http.Server{
			Addr: addr,
			//TODO: implement ServeMux
			// Handler: keystore.ServeMux(service),
		}

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error listening", err)
			os.Exit(1)
		}

		//TODO: add tls config

		go func() {
			err := server.Serve(ln)
			if err != nil {
				panic(err)
			}
		}()
		fmt.Fprintln(os.Stdout, "Server listening at https://localhost"+addr)

		// block forever without using any resources so this process won't quit while
		// the goroutine containing ListenAndServe is still working
		select {}
	}
}
