package server

import (
	"github.com/stellar/go/services/bifrost/sse"
)

func (s *Server) onStellarAccountCreated(destination string) {
	association, err := s.Database.GetAssociationByStellarPublicKey(destination)
	if err != nil {
		s.log.WithField("err", err).Error("Error getting association")
		return
	}

	if association == nil {
		s.log.WithField("stellarPublicKey", destination).Error("Association not found")
		return
	}

	s.SSEServer.BroadcastEvent(association.Address, sse.AccountCreatedAddressEvent, nil)
}

func (s *Server) onExchanged(destination string) {
	association, err := s.Database.GetAssociationByStellarPublicKey(destination)
	if err != nil {
		s.log.WithField("err", err).Error("Error getting association")
		return
	}

	if association == nil {
		s.log.WithField("stellarPublicKey", destination).Error("Association not found")
		return
	}

	s.SSEServer.BroadcastEvent(association.Address, sse.ExchangedEvent, nil)
}
