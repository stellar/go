// Skip this file in Go <1.8 because it's using http.Server.Shutdown
// +build go1.8

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	stdhttp "net/http"
	"os"
	"os/signal"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/bifrost/bitcoin"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/ethereum"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

func (s *Server) Start() error {
	s.initLogger()
	s.log.Info("Server starting")

	// Register callbacks
	s.BitcoinListener.TransactionHandler = s.onNewBitcoinTransaction
	s.EthereumListener.TransactionHandler = s.onNewEthereumTransaction
	s.StellarAccountConfigurator.OnAccountCreated = s.onStellarAccountCreated
	s.StellarAccountConfigurator.OnAccountCredited = s.onStellarAccountCredited

	if !s.BitcoinListener.Enabled && !s.EthereumListener.Enabled {
		return errors.New("At least one listener (BitcoinListener or EthereumListener) must be enabled")
	}

	if s.BitcoinListener.Enabled {
		var err error
		s.minimumValueSat, err = bitcoin.BtcToSat(s.MinimumValueBtc)
		if err != nil {
			return errors.Wrap(err, "Invalid minimum accepted Bitcoin transaction value: "+s.MinimumValueBtc)
		}

		if s.minimumValueSat == 0 {
			return errors.New("Minimum accepted Bitcoin transaction value must be larger than 0")
		}

		err = s.BitcoinListener.Start()
		if err != nil {
			return errors.Wrap(err, "Error starting BitcoinListener")
		}
	} else {
		s.log.Warn("BitcoinListener disabled")
	}

	if s.EthereumListener.Enabled {
		var err error
		s.minimumValueWei, err = ethereum.EthToWei(s.MinimumValueEth)
		if err != nil {
			return errors.Wrap(err, "Invalid minimum accepted Ethereum transaction value")
		}

		if s.minimumValueWei.Cmp(new(big.Int)) == 0 {
			return errors.New("Minimum accepted Ethereum transaction value must be larger than 0")
		}

		err = s.EthereumListener.Start(s.Config.Ethereum.RpcServer)
		if err != nil {
			return errors.Wrap(err, "Error starting EthereumListener")
		}
	} else {
		s.log.Warn("EthereumListener disabled")
	}

	err := s.StellarAccountConfigurator.Start()
	if err != nil {
		return errors.Wrap(err, "Error starting StellarAccountConfigurator")
	}

	err = s.SSEServer.StartPublishing()
	if err != nil {
		return errors.Wrap(err, "Error starting SSE Server")
	}

	signalInterrupt := make(chan os.Signal, 1)
	signal.Notify(signalInterrupt, os.Interrupt)

	go s.poolTransactionsQueue()
	go s.startHTTPServer()

	<-signalInterrupt
	s.shutdown()

	return nil
}

func (s *Server) initLogger() {
	s.log = common.CreateLogger("Server")
}

func (s *Server) shutdown() {
	if s.httpServer != nil {
		log.Info("Shutting down HTTP server...")
		ctx, close := context.WithTimeout(context.Background(), 5*time.Second)
		defer close()
		s.httpServer.Shutdown(ctx)
	}
}

func (s *Server) startHTTPServer() {
	mux := http.NewMux(s.Config.UsingProxy)

	mux.Get("/events", s.HandlerEvents)
	mux.Post("/generate-bitcoin-address", s.HandlerGenerateBitcoinAddress)
	mux.Post("/generate-ethereum-address", s.HandlerGenerateEthereumAddress)
	mux.Post("/recovery-transaction", s.HandlerRecoveryTransaction)

	addr := fmt.Sprintf("0.0.0.0:%d", s.Config.Port)

	http.Run(http.Config{
		ListenAddr: addr,
		Handler:    mux,
		OnStarting: func() {
			log.Infof("starting bifrost server - %s", app.Version())
			log.Infof("listening on %s", addr)
		},
	})
}

func (s *Server) HandlerEvents(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	// Create SSE stream if not exists but only if address exists.
	// This is required to restart a stream after server restart or failure.
	address := r.URL.Query().Get("stream")
	if !s.SSEServer.StreamExists(address) {
		var chain database.Chain

		if len(address) == 0 {
			w.WriteHeader(stdhttp.StatusBadRequest)
			return
		}

		if address[0] == '0' {
			chain = database.ChainEthereum
		} else {
			// 1 or m, n in testnet
			chain = database.ChainBitcoin
		}

		association, err := s.Database.GetAssociationByChainAddress(chain, address)
		if err != nil {
			log.WithField("err", err).Error("Error getting address association")
			w.WriteHeader(stdhttp.StatusInternalServerError)
			return
		}

		if association != nil {
			s.SSEServer.CreateStream(address)
		}
	}

	s.SSEServer.HTTPHandler(w, r)
}

func (s *Server) HandlerGenerateBitcoinAddress(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	s.handlerGenerateAddress(w, r, database.ChainBitcoin)
}

func (s *Server) HandlerGenerateEthereumAddress(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	s.handlerGenerateAddress(w, r, database.ChainEthereum)
}

func (s *Server) handlerGenerateAddress(w stdhttp.ResponseWriter, r *stdhttp.Request, chain database.Chain) {
	w.Header().Set("Access-Control-Allow-Origin", s.Config.AccessControlAllowOriginHeader)

	stellarPublicKey := r.PostFormValue("stellar_public_key")
	_, err := keypair.Parse(stellarPublicKey)
	if err != nil || (err == nil && stellarPublicKey[0] != 'G') {
		log.WithField("stellarPublicKey", stellarPublicKey).Warn("Invalid stellarPublicKey")
		w.WriteHeader(stdhttp.StatusBadRequest)
		return
	}

	index, err := s.Database.IncrementAddressIndex(chain)
	if err != nil {
		log.WithField("err", err).Error("Error incrementing address index")
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}

	var address string

	switch chain {
	case database.ChainBitcoin:
		address, err = s.BitcoinAddressGenerator.Generate(index)
	case database.ChainEthereum:
		address, err = s.EthereumAddressGenerator.Generate(index)
	default:
		log.WithField("chain", chain).Error("Invalid chain")
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}

	if err != nil {
		log.WithFields(log.F{"err": err, "index": index}).Error("Error generating address")
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}

	err = s.Database.CreateAddressAssociation(chain, stellarPublicKey, address, index)
	if err != nil {
		log.WithFields(log.F{
			"err":              err,
			"chain":            chain,
			"index":            index,
			"stellarPublicKey": stellarPublicKey,
			"address":          address,
		}).Error("Error creating address association")
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}

	// Create SSE stream
	s.SSEServer.CreateStream(address)

	response := GenerateAddressResponse{
		ProtocolVersion: ProtocolVersion,
		Chain:           string(chain),
		Address:         address,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.WithField("err", err).Error("Error encoding JSON")
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}

	w.Write(responseBytes)
}

func (s *Server) HandlerRecoveryTransaction(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	w.Header().Set("Access-Control-Allow-Origin", s.Config.AccessControlAllowOriginHeader)
	var transactionEnvelope xdr.TransactionEnvelope
	transactionXdr := r.PostFormValue("transaction_xdr")
	localLog := log.WithField("transaction_xdr", transactionXdr)

	if transactionXdr == "" {
		localLog.Warn("Invalid input. No Transaction XDR")
		w.WriteHeader(stdhttp.StatusBadRequest)
		return
	}

	err := xdr.SafeUnmarshalBase64(transactionXdr, &transactionEnvelope)
	if err != nil {
		localLog.WithField("err", err).Warn("Invalid Transaction XDR")
		w.WriteHeader(stdhttp.StatusBadRequest)
		return
	}

	err = s.Database.AddRecoveryTransaction(transactionEnvelope.Tx.SourceAccount.Address(), transactionXdr)
	if err != nil {
		localLog.WithField("err", err).Error("Error saving recovery transaction")
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}

	w.WriteHeader(stdhttp.StatusOK)
	return
}
