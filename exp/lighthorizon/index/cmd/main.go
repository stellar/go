package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

var (
	mutex     sync.RWMutex
	indexes   = map[string]*index.CheckpointIndex{}
	processed = uint64(0)
)

func main() {
	historyArchive, err := historyarchive.Connect(
		// "file:///Users/Bartek/archive",
		"s3://history.stellar.org/prd/core-live/core_live_001",
		historyarchive.ConnectOptions{
			NetworkPassphrase: network.PublicNetworkPassphrase,
			S3Region:          "eu-west-1",
			UnsignedRequests:  false,
		},
	)
	if err != nil {
		panic(err)
	}

	startTime := time.Now()

	startCheckpoint := uint32(0) //uint32((39684056) / 64)
	endCheckpoint := uint32((100000) / 64)
	all := endCheckpoint - startCheckpoint

	parallel := uint32(10)
	var wg sync.WaitGroup

	ch := make(chan uint32, 10)

	go func() {
		for i := startCheckpoint; i <= endCheckpoint; i++ {
			ch <- i
		}
		close(ch)
	}()

	for i := uint32(0); i < parallel; i++ {
		wg.Add(1)
		go func(i uint32) {
			for {
				checkpoint, ok := <-ch
				if !ok {
					wg.Done()
					return
				}

				if processed%20 == 0 {
					mutex.RLock()
					fmt.Printf(
						"Reading checkpoints... %d - %.2f%% - time elapsed: %s\n",
						checkpoint,
						(float64(processed)/float64(all))*100,
						time.Since(startTime),
					)
					mutex.RUnlock()
				}

				startLedger := checkpoint * 64
				if startLedger == 0 {
					startLedger = 1
				}
				endLedger := checkpoint*64 - 1 + 64

				fmt.Println("Processing checkpoint", checkpoint, "ledgers", startLedger, endLedger)

				ledgers, err := historyArchive.GetLedgers(startLedger, endLedger)
				if err != nil {
					panic(err)
				}

				for i := startLedger; i <= endLedger; i++ {
					ledger, ok := ledgers[i]
					if !ok {
						panic(fmt.Sprintf("no ledger %d", i))
					}

					resultMeta := make([]xdr.TransactionResultMeta, len(ledger.TransactionResult.TxResultSet.Results))
					for i, result := range ledger.TransactionResult.TxResultSet.Results {
						resultMeta[i].Result = result
					}

					closeMeta := xdr.LedgerCloseMeta{
						V0: &xdr.LedgerCloseMetaV0{
							LedgerHeader: ledger.Header,
							TxSet:        ledger.Transaction.TxSet,
							TxProcessing: resultMeta,
						},
					}

					reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, closeMeta)
					if err != nil {
						panic(err)
					}

					for {
						tx, err := reader.Read()
						if err != nil {
							if err == io.EOF {
								break
							}
							panic(err)
						}

						allParticipants, err := participantsForOperations(tx, false)
						if err != nil {
							panic(err)
						}

						addParticipantsToIndexes(checkpoint, "%s_all_all", allParticipants)

						paymentsParticipants, err := participantsForOperations(tx, true)
						if err != nil {
							panic(err)
						}

						addParticipantsToIndexes(checkpoint, "%s_all_payments", paymentsParticipants)

						if tx.Result.Successful() {
							allParticipants, err := participantsForOperations(tx, false)
							if err != nil {
								panic(err)
							}

							addParticipantsToIndexes(checkpoint, "%s_successful_all", allParticipants)

							paymentsParticipants, err := participantsForOperations(tx, true)
							if err != nil {
								panic(err)
							}

							addParticipantsToIndexes(checkpoint, "%s_successful_payments", paymentsParticipants)
						}
					}
				}

				mutex.Lock()
				processed++
				mutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	written := float32(0)
	for id, index := range indexes {
		err := os.WriteFile(fmt.Sprintf("./index/%s", id), index.Flush(), 0666)
		if err != nil {
			panic(err)
		}
		written++

		fmt.Printf("Writing indexes... %.2f%%\n", (written/float32(len(indexes)))*100)
	}
}

func addParticipantsToIndexes(checkpoint uint32, indexFormat string, participants []string) {
	for _, participant := range participants {
		ind := getCreateIndex(fmt.Sprintf(indexFormat, participant))
		err := ind.SetActive(checkpoint)
		if err != nil {
			panic(err)
		}
	}
}

func getCreateIndex(name string) *index.CheckpointIndex {
	mutex.Lock()
	defer mutex.Unlock()
	ind, ok := indexes[name]
	if !ok {
		ind = &index.CheckpointIndex{}
		indexes[name] = ind
	}
	return ind
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
