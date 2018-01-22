package stellar

import (
	"context"
	"net/http"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

const (
	// NewAccountXLMBalance is amount of lumens sent to new accounts
	NewAccountXLMBalance    = "41"
	defaultAccountPullDelay = 2 * time.Second
)

func (ac *AccountConfigurator) Start() error {
	ac.log = common.CreateLogger("StellarAccountConfigurator")
	ac.log.Info("StellarAccountConfigurator starting")

	kp, err := keypair.Parse(ac.IssuerPublicKey)
	if err != nil || (err == nil && ac.IssuerPublicKey[0] != 'G') {
		err = errors.Wrap(err, "Invalid IssuerPublicKey")
		ac.log.Error(err)
		return err
	}

	kp, err = keypair.Parse(ac.SignerSecretKey)
	if err != nil || (err == nil && ac.SignerSecretKey[0] != 'S') {
		err = errors.Wrap(err, "Invalid SignerSecretKey")
		ac.log.Error(err)
		return err
	}

	ac.signerPublicKey = kp.Address()

	root, err := ac.Horizon.Root()
	if err != nil {
		err = errors.Wrap(err, "Error loading Horizon root")
		ac.log.Error(err)
		return err
	}

	if root.NetworkPassphrase != ac.NetworkPassphrase {
		return errors.Errorf("Invalid network passphrase (have=%s, want=%s)", root.NetworkPassphrase, ac.NetworkPassphrase)
	}

	err = ac.updateSequence()
	if err != nil {
		err = errors.Wrap(err, "Error loading issuer sequence number")
		ac.log.Error(err)
		return err
	}

	go ac.logStats()
	return nil
}

func (ac *AccountConfigurator) logStats() {
	for {
		ac.log.WithField("currently_processing", ac.processingCount).Info("Stats")
		time.Sleep(15 * time.Second)
	}
}

func (ac *AccountConfigurator) activeProcessingCounter(delta int) {
	ac.processingCountMutex.Lock()
	ac.processingCount += delta
	ac.processingCountMutex.Unlock()
}

// ConfigureAccount configures a new account that participated in ICO.
// * First it creates a new account.
// * Once a trust line exists, it credits it with received number of ETH or BTC.
func (ac *AccountConfigurator) ConfigureAccount(ctx context.Context, transactionID, destination, assetCode, amount string) error {
	localLog := ac.log.WithFields(log.F{
		"destination": destination,
		"assetCode":   assetCode,
		"amount":      amount,
	})
	localLog.Info("Configuring Stellar account")

	ac.activeProcessingCounter(1)
	defer ac.activeProcessingCounter(-1)

	destAccount, destAccountExists, err := ac.getAccount(ctx, destination)
	if err != nil {
		return errors.Wrap(err, "failed to load account from Horizon")
	}

	if !destAccountExists {
		localLog.WithField("destination", destination).Info("Creating Stellar account")
		if err := ac.doCreateAccount(transactionID, assetCode, destination); err != nil {
			return err
		}
	}
	if !ac.trustlineExists(destAccount, assetCode) {
		if err := ac.awaitTrustLine(ctx, transactionID, assetCode, destination); err != nil {
			return err
		}
	}
	localLog.Info("Trust line found")

	// When trust line found check if needs to authorize, then send token
	if ac.NeedsAuthorize {
		localLog.Info("Authorizing trust line")
		if err := ac.allowTrust(destination, assetCode, ac.TokenAssetCode); err != nil {
			return errors.Wrap(err, "authorizing trust line failed")
		}
	}
	localLog.Info("Sending token")
	if err := ac.doSendTokens(ctx, transactionID, assetCode, destination, amount); err != nil {
		return err
	}
	localLog.Info("Account successfully configured")
	return nil
}

func (ac *AccountConfigurator) getAccount(ctx context.Context, account string) (horizon.Account, bool, error) {
	var hAccount horizon.Account
	if err := ctx.Err(); err != nil {
		return hAccount, false, err
	}
	result := make(chan error)
	go func() {
		defer close(result)
		var err error
		hAccount, err = ac.Horizon.LoadAccount(account)
		result <- err
	}()
	select {
	case err := <-result:
		if err == nil {
			return hAccount, true, nil
		}
		if err, ok := err.(*horizon.Error); ok && err.Response.StatusCode == http.StatusNotFound {
			return hAccount, false, nil
		}
		return hAccount, false, err
	case <-ctx.Done():
		return hAccount, false, ctx.Err()
	}
}

func (ac *AccountConfigurator) doCreateAccount(transactionID, assetCode, destination string) error {
	xdr, err := ac.SubmissionArchive.Find(transactionID, assetCode, SubmissionTypeCreateAccount)
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to find persisted submission")
	case err == nil && xdr != "":
		if err := ac.submitArchivedXDR(transactionID, assetCode, SubmissionTypeCreateAccount, xdr); err != nil {
			return err
		}
	default:
		if err := ac.createAccount(transactionID, assetCode, destination); err != nil {
			return errors.Wrap(err, "failed to create Stellar account")
		}
	}
	if ac.OnAccountCreated != nil {
		ac.OnAccountCreated(destination)
	}
	return nil
}

func (ac *AccountConfigurator) awaitTrustLine(ctx context.Context, transactionID, assetCode, destination string) error {
	// Wait for trust line to be created...
	retryDelayer := time.NewTimer(defaultAccountPullDelay)
	defer retryDelayer.Stop()
	exit := ctx.Done()
	for i := 0; i < 10; i++ {
		destAccount, destAccountExists, err := ac.getAccount(ctx, destination)
		if err != nil {
			return errors.Wrap(err, "failed to load account to check trust line")
		}
		if destAccountExists && ac.trustlineExists(destAccount, assetCode) {
			return nil
		}
		retryDelayer.Reset(defaultAccountPullDelay)
		select {
		case <-exit:
			return ctx.Err()
		case <-retryDelayer.C:
			continue
		}
	}
	return errors.New("failed to find trust line set")
}

func (ac *AccountConfigurator) doSendTokens(ctx context.Context, transactionID, assetCode, destination, amount string) error {
	xdr, err := ac.SubmissionArchive.Find(transactionID, assetCode, SubmissionTypeSendTokens)
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to find persisted submission")
	case ctx.Err() != nil:
		return ctx.Err()
	case err == nil && xdr != "":
		if err := ac.submitArchivedXDR(transactionID, assetCode, SubmissionTypeSendTokens, xdr); err != nil {
			return err
		}
	default:
		if err := ac.sendToken(transactionID, assetCode, destination, amount); err != nil {
			return errors.Wrap(err, "sending asset to account failed")
		}
	}

	if ac.OnAccountCredited != nil {
		ac.OnAccountCredited(destination, assetCode, amount)
	}
	return nil
}

func (ac *AccountConfigurator) trustlineExists(account horizon.Account, assetCode string) bool {
	for _, balance := range account.Balances {
		if balance.Asset.Issuer == ac.IssuerPublicKey && balance.Asset.Code == assetCode {
			return true
		}
	}

	return false
}
