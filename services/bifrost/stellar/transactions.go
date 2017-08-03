package stellar

import (
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (ac *AccountConfigurator) createAccount(destination string) error {
	err := ac.submitTransaction(
		build.CreateAccount(
			build.Destination{destination},
			build.NativeAmount{NewAccountXLMBalance},
		),
	)
	if err != nil {
		return errors.Wrap(err, "Error submitting transaction")
	}

	return nil
}

func (ac *AccountConfigurator) sendToken(destination, assetCode, amount string) error {
	err := ac.submitTransaction(
		build.Payment(
			build.Destination{destination},
			build.CreditAmount{
				Code:   assetCode,
				Issuer: ac.issuerPublicKey,
				Amount: amount,
			},
		),
	)
	if err != nil {
		return errors.Wrap(err, "Error submitting transaction")
	}

	return nil
}

func (ac *AccountConfigurator) submitTransaction(mutator build.TransactionMutator) error {
	tx, err := ac.buildTransaction(mutator)
	if err != nil {
		return errors.Wrap(err, "Error building transaction")
	}

	localLog := log.WithField("tx", tx)
	localLog.Info("Submitting transaction")

	_, err = ac.Horizon.SubmitTransaction(tx)
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

func (ac *AccountConfigurator) buildTransaction(mutator build.TransactionMutator) (string, error) {
	tx := build.Transaction(
		build.SourceAccount{ac.IssuerSecretKey},
		build.Sequence{ac.getSequence()},
		build.Network{ac.NetworkPassphrase},
		mutator,
	)

	txe := tx.Sign(ac.IssuerSecretKey)
	return txe.Base64()
}
