package stellar

import (
	"strconv"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (ac *AccountConfigurator) createAccountTransaction(destination string) error {
	transaction, err := ac.buildTransaction(
		[]string{ac.SignerSecretKey},
		build.CreateAccount(
			build.SourceAccount{ac.IssuerPublicKey},
			build.Destination{destination},
			build.NativeAmount{ac.StartingBalance},
		),
	)
	if err != nil {
		return errors.Wrap(err, "Error building transaction")
	}

	err = ac.submitTransaction(transaction)
	if err != nil {
		return errors.Wrap(err, "Error submitting a transaction")
	}

	return nil
}

// configureAccountTransaction is using a temporary signer on an user accounts to configure the account.
func (ac *AccountConfigurator) configureAccountTransaction(destination, intermediateAssetCode, amount string, needsAuthorize bool) error {
	mutators := []build.TransactionMutator{
		build.Trust(intermediateAssetCode, ac.TokenAssetCode),
		build.Trust(ac.TokenAssetCode, ac.TokenAssetCode),
	}

	if needsAuthorize {
		mutators = append(
			mutators,
			// Chain token received (BTC/ETH)
			build.AllowTrust(
				build.SourceAccount{ac.IssuerPublicKey},
				build.Trustor{destination},
				build.AllowTrustAsset{intermediateAssetCode},
				build.Authorize{true},
			),
			// Destination token
			build.AllowTrust(
				build.SourceAccount{ac.IssuerPublicKey},
				build.Trustor{destination},
				build.AllowTrustAsset{ac.TokenAssetCode},
				build.Authorize{true},
			),
		)
	}

	createOffer := build.CreateOffer(
		build.Rate{
			Selling: build.CreditAsset(intermediateAssetCode, ac.IssuerPublicKey),
			Buying:  build.CreditAsset(ac.TokenAssetCode, ac.IssuerPublicKey),
			Price:   build.Price(ac.TokenPrice),
		},
		build.Amount(amount),
	)

	mutators = append(
		mutators,
		// Send BTC/ETH
		build.Payment(
			build.SourceAccount{ac.IssuerPublicKey},
			build.Destination{destination},
			build.CreditAmount{
				Code:   intermediateAssetCode,
				Issuer: ac.IssuerPublicKey,
				Amount: amount,
			},
		),
		// Exchange BTC/ETH => token
		createOffer,
	)

	transaction, err := ac.buildTransaction([]string{ac.SignerSecretKey, ac.TemporaryAccountSignerSecretKey}, mutators...)
	if err != nil {
		return errors.Wrap(err, "Error building a transaction")
	}

	err = ac.submitTransaction(transaction)
	if err != nil {
		return errors.Wrap(err, "Error submitting a transaction")
	}

	return nil
}

func (ac *AccountConfigurator) buildTransaction(signers []string, mutators ...build.TransactionMutator) (string, error) {
	muts := []build.TransactionMutator{
		build.SourceAccount{ac.signerPublicKey},
		build.Sequence{ac.getSequence()},
		build.Network{ac.NetworkPassphrase},
	}
	muts = append(muts, mutators...)
	tx, err := build.Transaction(muts...)
	if err != nil {
		return "", err
	}
	txe, err := tx.Sign(signers...)
	if err != nil {
		return "", err
	}
	return txe.Base64()
}

func (ac *AccountConfigurator) submitTransaction(transaction string) error {
	localLog := log.WithField("tx", transaction)
	localLog.Info("Submitting transaction")

	_, err := ac.Horizon.SubmitTransaction(transaction)
	if err != nil {
		fields := log.F{"err": err}
		if err, ok := err.(*horizon.Error); ok {
			fields["result"] = string(err.Problem.Extras["result_xdr"])
			ac.updateSequence()
		}
		localLog.WithFields(fields).Error("Error submitting transaction")
		return errors.Wrap(err, "Error submitting transaction")
	}

	localLog.Info("Transaction successfully submitted")
	return nil
}

func (ac *AccountConfigurator) updateSequence() error {
	ac.sequenceMutex.Lock()
	defer ac.sequenceMutex.Unlock()

	account, err := ac.Horizon.LoadAccount(ac.signerPublicKey)
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
