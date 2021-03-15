package assetissuer

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
)

type Options struct {
	HorizonURL          string
	NetworkPassphrase   string
	AccountIssuerSecret string
}

func (opts Options) horizonClient() horizonclient.ClientInterface {
	var client *horizonclient.Client
	if opts.NetworkPassphrase == network.PublicNetworkPassphrase {
		client = horizonclient.DefaultPublicNetClient
	} else {
		client = horizonclient.DefaultTestNetClient
	}

	client.HorizonURL = opts.HorizonURL

	return client
}

// Configure will set the account flags needed for regulated assets to
// "auth_required: true" and "auth_revocable: true".
// ref1:https://developers.stellar.org/docs/issuing-assets/control-asset-access/
// ref2:https://github.com/stellar/stellar-protocol/blob/d49e04af8e047474f2c506d9d11bb63b6ad55d2c/ecosystem/sep-0008.md#authorization-flags
func Configure(opts Options) {
	kp, err := keypair.ParseFull(opts.AccountIssuerSecret)
	if err != nil {
		log.Fatal(errors.Wrap(err, "parsing secret"))
	}

	horizonClient := opts.horizonClient()

	log.DefaultLogger.Infof("Account address: %s\n", kp.Address())
	account, err := horizonClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: kp.Address(),
	})
	if err != nil {
		log.Fatal(errors.Wrap(err, "getting account detail"))
	}

	if account.Flags.AuthRevocable && account.Flags.AuthRequired {
		log.Info("Account flags already contain \"Auth Required\" and \"Auth Revocable\"")
		return
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &account,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.SetOptions{
					SetFlags: []txnbuild.AccountFlag{
						txnbuild.AuthRequired,
						txnbuild.AuthRevocable,
					},
				},
			},
			BaseFee:    300,
			Timebounds: txnbuild.NewTimeout(300),
		},
	)
	if err != nil {
		log.Fatal(errors.Wrap(err, "creating transaction"))
	}

	tx, err = tx.Sign(opts.NetworkPassphrase, kp)
	if err != nil {
		log.Fatal(errors.Wrap(err, "signing transaction"))
	}

	log.Info("Will update account flags")
	_, err = horizonClient.SubmitTransaction(tx)
	if err != nil {
		log.Fatal(parseHorizonError(err))
	}
	log.Info("Did update account flags")
}

func parseHorizonError(err error) error {
	if err == nil {
		return nil
	}

	rootErr := errors.Cause(err)
	if hError := horizonclient.GetError(rootErr); hError != nil {
		resultCode, _ := hError.ResultCodes()
		err = errors.Wrapf(err, "error submitting transaction: %+v, %+v\n", hError.Problem, resultCode)
	} else {
		err = errors.Wrap(err, "error submitting transaction")
	}
	return err
}
