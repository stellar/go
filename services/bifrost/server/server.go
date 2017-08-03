package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/r3labs/sse"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (s *Server) Start() error {
	s.log = common.CreateLogger("Server")
	s.log.Info("Server starting")

	// Register callbacks
	s.EthereumListener.TransactionHandler = s.onNewEthereumTransaction
	s.StellarAccountConfigurator.OnAccountCreated = s.onStellarAccountCreated
	s.StellarAccountConfigurator.OnAccountCredited = s.onStellarAccountCredited

	err := s.EthereumListener.Start(s.Config.Ethereum.RpcServer)
	if err != nil {
		return errors.Wrap(err, "Error starting EthereumListener")
	}

	err = s.StellarAccountConfigurator.Start()
	if err != nil {
		return errors.Wrap(err, "Error starting StellarAccountConfigurator")
	}

	go s.poolTransactionsQueue()

	s.startHTTPServer()
	return nil
}

func (s *Server) startHTTPServer() {
	s.eventsServer = sse.New()

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/events", s.eventsServer.HTTPHandler)
	r.Post("/generate-ethereum-address", s.handlerGenerateEthereumAddress)

	log.Info("Starting HTTP server")
	// TODO read from config
	err := http.ListenAndServe(":3000", r)
	if err != nil {
		log.WithField("err", err).Fatal("Cannot start HTTP server")
	}
}

func (s *Server) handlerGenerateEthereumAddress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stellarPublicKey := r.PostFormValue("stellar_public_key")
	// TODO validation and check if already exists

	index, err := s.Database.IncrementEthereumAddressIndex()
	if err != nil {
		log.WithField("err", err).Error("Error incrementing ethereum address index")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	address, err := s.EthereumAddressGenerator.Generate(index)
	if err != nil {
		log.WithFields(log.F{"err": err, "index": index}).Error("Error generating ethereum address")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.Database.CreateEthereumAddressAssociation(stellarPublicKey, address, index)
	if err != nil {
		log.WithFields(log.F{"err": err, "index": index}).Error("Error creating ethereum address association")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create SSE stream
	s.eventsServer.CreateStream(address)

	response := GenerateEthereumAddressResponse{address}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.WithField("err", err).Error("Error encoding JSON")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(responseBytes)
}

func (s *Server) publishEvent(address string, event AddressEvent, data []byte) {
	// Create SSE stream if not exists
	if !s.eventsServer.StreamExists(address) {
		s.eventsServer.CreateStream(address)
	}

	// github.com/r3labs/sse does not send new lines - TODO create PR
	if data == nil {
		data = []byte("{}\n")
	} else {
		data = append(data, byte('\n'))
	}

	s.eventsServer.Publish(address, &sse.Event{
		Event: []byte(event),
		Data:  data,
	})
}
