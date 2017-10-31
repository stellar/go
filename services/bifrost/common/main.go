package common

import (
	"github.com/stellar/go/support/log"
)

const StellarAmountPrecision = 7

func CreateLogger(serviceName string) *log.Entry {
	logger := log.DefaultLogger

	if serviceName != "" {
		logger = logger.WithField("service", serviceName)
	}

	return logger
}
