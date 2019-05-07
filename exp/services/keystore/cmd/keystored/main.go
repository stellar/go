package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/stellar/go/exp/services/keystore"
	"github.com/stellar/go/support/log"

	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()
	tlsCert := flag.String("tls-cert", "", "TLS certificate file path")
	tlsKey := flag.String("tls-key", "", "TLS private key file path")

	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "too few arguments")
		os.Exit(1)
	}

	if (*tlsCert == "" && *tlsKey != "") || (*tlsCert != "" && *tlsKey == "") {
		fmt.Fprintln(os.Stderr, "TLS cert and TLS key have to be presented together")
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

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error listening", err)
			os.Exit(1)
		}

		listener = tcpKeepAliveListener{listener.(*net.TCPListener)}
		if *tlsCert != "" {
			cer, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error parsing TLS keypair", err)
				os.Exit(1)
			}

			listener = tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{cer}})
		}

		go func() {
			err := server.Serve(listener)
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

// https://github.com/golang/go/blob/c5cf6624076a644906aa7ec5c91c4e01ccd375d3/src/net/http/server.go#L3272-L3288
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
