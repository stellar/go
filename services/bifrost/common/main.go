package common

import (
	"math/big"

	"github.com/stellar/go/support/log"
)

const StellarAmountPrecision = 7

var (
	eight    = big.NewInt(8)
	ten      = big.NewInt(10)
	eighteen = big.NewInt(18)
	// weiInEth = 10^18
	weiInEth = new(big.Rat).SetInt(new(big.Int).Exp(ten, eighteen, nil))
	// satInBtc = 10^8
	satInBtc = new(big.Rat).SetInt(new(big.Int).Exp(ten, eight, nil))
)

func CreateLogger(serviceName string) *log.Entry {
	logger := log.DefaultLogger

	if serviceName != "" {
		logger = logger.WithField("service", serviceName)
	}

	return logger
}
