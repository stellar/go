package stellar

import (
	"net/http"
	"strconv"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// NewAccountXLMBalance is amount of lumens sent to new accounts
const NewAccountXLMBalance = "41"

func (ac *AccountConfigurator) Start() error {
	ac.log = common.CreateLogger("StellarAccountConfigurator")
	ac.log.Info("StellarAccountConfigurator starting")

	kp, err := keypair.Parse(ac.IssuerSecretKey)
	if err != nil {
		err = errors.Wrap(err, "Invalid IssuerSecretKey")
		ac.log.Error(err)
		return err
	}

	ac.issuerPublicKey = kp.Address()

	err = ac.updateSequence()
	if err != nil {
		err = errors.Wrap(err, "Error loading issuer sequence number")
		ac.log.Error(err)
		return err
	}

	return nil
}

// ConfigureAccount configures a new account that participated in ICO.
// * First it creates a new account.
// * Once a trusline exists, it credits it with sent number of ETH or BTC.
func (ac *AccountConfigurator) ConfigureAccount(destination, assetCode, amount string) {
	localLog := ac.log.WithFields(log.F{
		"destination": destination,
		"assetCode":   assetCode,
		"amount":      amount,
	})
	localLog.Info("Configuring Stellar account")

	// Check if account exists. If it is, skip creating it.
	_, exists, err := ac.getAccount(destination)
	if err != nil {
		localLog.WithField("err", err).Error("Error loading account from Horizon")
		return
	}

	if !exists {
		localLog.WithField("destination", destination).Info("Creating Stellar account")
		err := ac.createAccount(destination)
		if err != nil {
			localLog.WithField("err", err).Error("Error creating Stellar account")
			// TODO repeat
			return
		}
	}

	if ac.OnAccountCreated != nil {
		ac.OnAccountCreated(destination)
	}

	// TODO if exists but native balance is too small, send more XLM?

	// Wait for account and trustline to be created...
	for {
		account, err := ac.Horizon.LoadAccount(destination)
		if err != nil {
			localLog.WithField("err", err).Error("Error loading account to check trustline")
			time.Sleep(2 * time.Second)
			continue
		}

		if ac.trustlineExists(account, assetCode) {
			break
		} else {
			time.Sleep(2 * time.Second)
		}
	}

	// When trustline found send token
	localLog.Info("Trust line found, sending token")
	err = ac.sendToken(destination, assetCode, amount)
	if err != nil {
		localLog.WithField("err", err).Error("Error sending asset to account")
		return
	}

	if ac.OnAccountCredited != nil {
		ac.OnAccountCredited(destination, assetCode, amount)
	}

	localLog.Info("Account successully configured")
}

func (ac *AccountConfigurator) getAccount(account string) (horizon.Account, bool, error) {
	var hAccount horizon.Account
	hAccount, err := ac.Horizon.LoadAccount(account)
	if err != nil {
		if err, ok := err.(*horizon.Error); ok && err.Response.StatusCode == http.StatusNotFound {
			return hAccount, false, nil
		}
		return hAccount, false, err
	}

	return hAccount, true, nil
}

func (ac *AccountConfigurator) trustlineExists(account horizon.Account, assetCode string) bool {
	for _, balance := range account.Balances {
		if balance.Asset.Issuer == ac.issuerPublicKey && balance.Asset.Code == assetCode {
			return true
		}
	}

	return false
}

func (ac *AccountConfigurator) updateSequence() error {
	ac.sequenceMutex.Lock()
	defer ac.sequenceMutex.Unlock()

	account, err := ac.Horizon.LoadAccount(ac.issuerPublicKey)
	if err != nil {
		err = errors.Wrap(err, "Error loading issuing account")
		ac.log.Error(err)
		return err
	}

	ac.sequence, err = strconv.ParseUint(account.Sequence, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "Invalid account.Sequence")
		ac.log.Error(err)
		return err
	}

	return nil
}

func (ac *AccountConfigurator) getSequence() uint64 {
	ac.sequenceMutex.Lock()
	defer ac.sequenceMutex.Unlock()
	ac.sequence++
	sequence := ac.sequence
	return sequence
}
