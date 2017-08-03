package server

import (
	"encoding/json"
)

func (s *Server) onStellarAccountCreated(destination string) {
	association, err := s.Database.GetAssociationByStellarPublicKey(destination)
	if err != nil {
		s.log.WithField("err", err).Error("Error getting association")
		return
	}

	if association == nil {
		s.log.WithField("stellarPublicKey", destination).Warn("Association not found")
		return
	}

	s.publishEvent(association.Address, AccountCreatedAddressEvent, nil)
}

func (s *Server) onStellarAccountCredited(destination, assetCode, amount string) {
	association, err := s.Database.GetAssociationByStellarPublicKey(destination)
	if err != nil {
		s.log.WithField("err", err).Error("Error getting association")
		return
	}

	if association == nil {
		s.log.WithField("stellarPublicKey", destination).Warn("Association not found")
		return
	}

	data := map[string]string{
		"assetCode": assetCode,
		"amount":    amount,
	}

	j, err := json.Marshal(data)
	if err != nil {
		s.log.WithField("data", data).Error("Error marshalling json")
	}

	s.publishEvent(association.Address, AccountCreditedAddressEvent, j)
}
