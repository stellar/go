package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
	"golang.org/x/sync/errgroup"
)

var (
	// Should we use runtime.NumCPU() for a reasonable default?
	parallel = uint32(20)
)

func main() {
	sourceUrl := flag.String("source", "gcs://horizon-archive-poc", "history archive url to read txmeta files")
	targetUrl := flag.String("target", "file://indexes", "where to write indexes")
	networkPassphrase := flag.String("network-passphrase", network.TestNetworkPassphrase, "network passphrase")
	start := flag.Int("start", -1, "ledger to start at (default: earliest)")
	end := flag.Int("end", -1, "ledger to end at (default: latest)")
	modules := flag.String("modules", "accounts,transactions", "comma-separated list of modules to index (default: all)")
	flag.Parse()

	log.SetLevel(log.InfoLevel)

	ctx := context.Background()

	indexStore, err := index.Connect(*targetUrl)
	if err != nil {
		panic(err)
	}

	// Simple file os access
	source, err := historyarchive.ConnectBackend(
		*sourceUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: *networkPassphrase,
		},
	)
	if err != nil {
		panic(err)
	}
	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(source)
	defer ledgerBackend.Close()

	startTime := time.Now()

	if *start < 2 {
		*start = 2
	}
	if *end == -1 {
		latest, err := ledgerBackend.GetLatestLedgerSequence(ctx)
		if err != nil {
			panic(err)
		}
		*end = int(latest)
	}
	startLedger := uint32(*start) //uint32((39680056) / 64)
	endLedger := uint32(*end)
	all := endLedger - startLedger

	wg, ctx := errgroup.WithContext(ctx)

	ch := make(chan uint32, parallel)

	go func() {
		for i := startLedger; i <= endLedger; i++ {
			ch <- i
		}
		close(ch)
	}()

	processed := uint64(0)
	for i := uint32(0); i < parallel; i++ {
		wg.Go(func() error {
			for ledgerSeq := range ch {
				fmt.Println("Processing ledger", ledgerSeq)
				ledger, err := ledgerBackend.GetLedger(ctx, ledgerSeq)
				if err != nil {
					log.WithField("error", err).Error("error getting ledgers")
					ch <- ledgerSeq
					continue
				}

				checkpoint := ledgerSeq / 64

				reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(*networkPassphrase, ledger)
				if err != nil {
					return err
				}

				for {
					tx, err := reader.Read()
					if err != nil {
						if err == io.EOF {
							break
						}
						return err
					}

					if strings.Contains(*modules, "transactions") {
						indexStore.AddTransactionToIndexes(
							toid.New(int32(ledger.LedgerSequence()), int32(tx.Index), 0).ToInt64(),
							tx.Result.TransactionHash,
						)
					}


					if strings.Contains(*modules, "accounts") {
						allParticipants, err := participantsForOperations(tx, false)
						if err != nil {
							return err
						}

						err = indexStore.AddParticipantsToIndexes(checkpoint, "all_all", allParticipants)
						if err != nil {
							return err
						}

						paymentsParticipants, err := participantsForOperations(tx, true)
						if err != nil {
							return err
						}

						err = indexStore.AddParticipantsToIndexes(checkpoint, "all_payments", paymentsParticipants)
						if err != nil {
							return err
						}

						if tx.Result.Successful() {
							allParticipants, err := participantsForOperations(tx, false)
							if err != nil {
								return err
							}

							err = indexStore.AddParticipantsToIndexes(checkpoint, "successful_all", allParticipants)
							if err != nil {
								return err
							}

							paymentsParticipants, err := participantsForOperations(tx, true)
							if err != nil {
								return err
							}

							err = indexStore.AddParticipantsToIndexes(checkpoint, "successful_payments", paymentsParticipants)
							if err != nil {
								return err
							}
						}
					}
				}
			}

			nprocessed := atomic.AddUint64(&processed, 1)

			if nprocessed%100 == 0 {
				log.Infof(
					"Reading checkpoints... - %.2f%% - elapsed: %s, remaining: %s",
					(float64(nprocessed)/float64(all))*100,
					time.Since(startTime).Round(1*time.Second),
					(time.Duration(int64(time.Since(startTime))*int64(all)/int64(nprocessed)) - time.Since(startTime)).Round(1*time.Second),
				)

				// Clear indexes to save memory
				if err := indexStore.Flush(); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		panic(err)
	}
	log.Infof("Uploading indexes")
	if err := indexStore.Flush(); err != nil {
		panic(err)
	}
}

func participantsForOperations(transaction ingest.LedgerTransaction, onlyPayments bool) ([]string, error) {
	var participants []string

	for opindex, operation := range transaction.Envelope.Operations() {
		opSource := operation.SourceAccount
		if opSource == nil {
			txSource := transaction.Envelope.SourceAccount()
			opSource = &txSource
		}

		switch operation.Body.Type {
		case xdr.OperationTypeCreateAccount,
			xdr.OperationTypePayment,
			xdr.OperationTypePathPaymentStrictReceive,
			xdr.OperationTypePathPaymentStrictSend,
			xdr.OperationTypeAccountMerge:
			participants = append(participants, opSource.Address())
		default:
			if onlyPayments {
				continue
			}
			participants = append(participants, opSource.Address())
		}

		switch operation.Body.Type {
		case xdr.OperationTypeCreateAccount:
			participants = append(participants, operation.Body.MustCreateAccountOp().Destination.Address())
		case xdr.OperationTypePayment:
			participants = append(participants, operation.Body.MustPaymentOp().Destination.ToAccountId().Address())
		case xdr.OperationTypePathPaymentStrictReceive:
			participants = append(participants, operation.Body.MustPathPaymentStrictReceiveOp().Destination.ToAccountId().Address())
		case xdr.OperationTypePathPaymentStrictSend:
			participants = append(participants, operation.Body.MustPathPaymentStrictSendOp().Destination.ToAccountId().Address())
		case xdr.OperationTypeManageBuyOffer:
			// the only direct participant is the source_account
		case xdr.OperationTypeManageSellOffer:
			// the only direct participant is the source_account
		case xdr.OperationTypeCreatePassiveSellOffer:
			// the only direct participant is the source_account
		case xdr.OperationTypeSetOptions:
			// the only direct participant is the source_account
		case xdr.OperationTypeChangeTrust:
			// the only direct participant is the source_account
		case xdr.OperationTypeAllowTrust:
			participants = append(participants, operation.Body.MustAllowTrustOp().Trustor.Address())
		case xdr.OperationTypeAccountMerge:
			participants = append(participants, operation.Body.MustDestination().ToAccountId().Address())
		case xdr.OperationTypeInflation:
			// the only direct participant is the source_account
		case xdr.OperationTypeManageData:
			// the only direct participant is the source_account
		case xdr.OperationTypeBumpSequence:
			// the only direct participant is the source_account
		case xdr.OperationTypeCreateClaimableBalance:
			for _, c := range operation.Body.MustCreateClaimableBalanceOp().Claimants {
				participants = append(participants, c.MustV0().Destination.Address())
			}
		case xdr.OperationTypeClaimClaimableBalance:
			// the only direct participant is the source_account
		case xdr.OperationTypeBeginSponsoringFutureReserves:
			participants = append(participants, operation.Body.MustBeginSponsoringFutureReservesOp().SponsoredId.Address())
		case xdr.OperationTypeEndSponsoringFutureReserves:
			// Failed transactions may not have a compliant sandwich structure
			// we can rely on (e.g. invalid nesting or a being operation with the wrong sponsoree ID)
			// and thus we bail out since we could return incorrect information.
			if transaction.Result.Successful() {
				sponsoree := transaction.Envelope.SourceAccount().ToAccountId().Address()
				if operation.SourceAccount != nil {
					sponsoree = operation.SourceAccount.Address()
				}
				operations := transaction.Envelope.Operations()
				for i := int(opindex) - 1; i >= 0; i-- {
					if beginOp, ok := operations[i].Body.GetBeginSponsoringFutureReservesOp(); ok &&
						beginOp.SponsoredId.Address() == sponsoree {
						participants = append(participants, beginOp.SponsoredId.Address())
					}
				}
			}
		case xdr.OperationTypeRevokeSponsorship:
			op := operation.Body.MustRevokeSponsorshipOp()
			switch op.Type {
			case xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry:
				participants = append(participants, getLedgerKeyParticipants(*op.LedgerKey)...)
			case xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner:
				participants = append(participants, op.Signer.AccountId.Address())
				// We don't add signer as a participant because a signer can be arbitrary account.
				// This can spam successful operations history of any account.
			}
		case xdr.OperationTypeClawback:
			op := operation.Body.MustClawbackOp()
			participants = append(participants, op.From.ToAccountId().Address())
		case xdr.OperationTypeClawbackClaimableBalance:
			// the only direct participant is the source_account
		case xdr.OperationTypeSetTrustLineFlags:
			op := operation.Body.MustSetTrustLineFlagsOp()
			participants = append(participants, op.Trustor.Address())
		case xdr.OperationTypeLiquidityPoolDeposit:
			// the only direct participant is the source_account
		case xdr.OperationTypeLiquidityPoolWithdraw:
			// the only direct participant is the source_account
		default:
			return nil, fmt.Errorf("unknown operation type: %s", operation.Body.Type)
		}

		// Requires meta
		// sponsor, err := operation.getSponsor()
		// if err != nil {
		// 	return nil, err
		// }
		// if sponsor != nil {
		// 	otherParticipants = append(otherParticipants, *sponsor)
		// }
	}

	return participants, nil
}

func getLedgerKeyParticipants(ledgerKey xdr.LedgerKey) []string {
	var result []string
	switch ledgerKey.Type {
	case xdr.LedgerEntryTypeAccount:
		result = append(result, ledgerKey.Account.AccountId.Address())
	case xdr.LedgerEntryTypeClaimableBalance:
		// nothing to do
	case xdr.LedgerEntryTypeData:
		result = append(result, ledgerKey.Data.AccountId.Address())
	case xdr.LedgerEntryTypeOffer:
		result = append(result, ledgerKey.Offer.SellerId.Address())
	case xdr.LedgerEntryTypeTrustline:
		result = append(result, ledgerKey.TrustLine.AccountId.Address())
	}
	return result
}
