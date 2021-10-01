package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/buffer"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"golang.org/x/sync/errgroup"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
	}
}

func run() (err error) {
	showHelp := false
	cpuProfileFile := ""
	memProfileFile := ""
	horizonURL := "https://horizon-testnet.stellar.org"
	count := 10_000
	sourceAccountSeed := "S..."
	destinationAccount := "G..."
	destinationAddr := ":8001"

	fs := flag.NewFlagSet("benchmark", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.BoolVar(&showHelp, "h", showHelp, "Show this help")
	fs.StringVar(&cpuProfileFile, "cpuprofile", cpuProfileFile, "Write cpu profile to `file`")
	fs.StringVar(&memProfileFile, "memprofile", memProfileFile, "Write memory profile to `file`")
	fs.StringVar(&horizonURL, "horizon", horizonURL, "Horizon URL")
	fs.IntVar(&count, "count", count, "Count of payments to send")
	fs.StringVar(&sourceAccountSeed, "source-account", sourceAccountSeed, "Source account to send payments from")
	fs.StringVar(&destinationAccount, "destination-account", destinationAccount, "Destination account to send payments to")
	fs.StringVar(&destinationAddr, "destination-addr", destinationAddr, "Address and port of the destination to post batch meta to")

	err = fs.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	if showHelp {
		fs.Usage()
		return nil
	}

	if cpuProfileFile != "" {
		f, err := os.Create(cpuProfileFile)
		if err != nil {
			return fmt.Errorf("error creating cpu profile file: %w", err)
		}
		defer f.Close()
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return fmt.Errorf("error starting cpu profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}
	if memProfileFile != "" {
		defer func() {
			if err != nil {
				return
			}
			var f *os.File
			f, err = os.Create(memProfileFile)
			if err != nil {
				err = fmt.Errorf("error creating memory profile file: %w", err)
				return
			}
			defer f.Close()
			runtime.GC()
			pprofErr := pprof.WriteHeapProfile(f)
			if pprofErr != nil {
				err = fmt.Errorf("error creating memory profile file: %w", pprofErr)
				return
			}
		}()
	}

	horizonClient := &horizonclient.Client{HorizonURL: horizonURL}
	networkDetails, err := horizonClient.Root()
	if err != nil {
		return fmt.Errorf("getting network details: %w", err)
	}
	fmt.Printf("Network: %s\n", networkDetails.NetworkPassphrase)

	if sourceAccountSeed == "S..." {
		sourceAccountSeed = keypair.MustRandom().Seed()
	}
	sourceAccountKey := keypair.MustParseFull(sourceAccountSeed)
	sourceAccount := sourceAccountKey.Address()
	fmt.Printf("Source: %s\n", sourceAccount)

	if destinationAccount == "G..." {
		destinationAccount = keypair.MustRandom().Address()
	}
	fmt.Printf("Destination: %s\n", destinationAccount)

	fmt.Printf("Funding... %s\n", sourceAccount)
	_, err = horizonClient.Fund(sourceAccount)
	if err != nil {
		fmt.Printf("Error funding source: %v\n", err)
	}
	fmt.Printf("Funding... %s\n", destinationAccount)
	_, err = horizonClient.Fund(destinationAccount)
	if err != nil {
		fmt.Printf("Error funding destination: %v\n", err)
	}

	stats := stats{}

	// Send payments.
	b := buffer.NewPaymentBuffer(buffer.PaymentBufferConfig{
		MaxBatchSize:    2_000_000,
		NewBatchIDFunc:  buffer.PaymentBufferNewBatchIDFunc(newBatchID),
		SubmitBatchFunc: submitBatchFunc(networkDetails.NetworkPassphrase, horizonClient, sourceAccountKey, destinationAccount, destinationAddr, &stats),
	})

	fmt.Printf("Starting...\n")
	stats.timeStarted = time.Now()
	for i := 0; i < count; i++ {
		for {
			_, err = b.Payment(buffer.PaymentParams{
				Amount: 1,
				Memo:   int64(i),
			})
			if errors.Is(err, buffer.ErrBatchFull) {
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
	}
	b.Wait()
	stats.timeFinished = time.Now()
	fmt.Printf("Finished...\n")

	fmt.Printf("%v\n", stats)

	b.Done()

	return nil
}

func submitBatchFunc(networkPassphrase string, horizonClient *horizonclient.Client, sourceAccountKey *keypair.Full, destinationAccount, destinationAddr string, s *stats) buffer.PaymentBufferSubmitBatchFunc {
	return func(batchID string, batchAmount int64, payments []buffer.PaymentParams) {
		a, err := horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: sourceAccountKey.Address()})
		if err != nil {
			fmt.Printf("Error getting source account details: %v\n", err)
			return
		}

		g := errgroup.Group{}
		g.Go(func() error {
			pr, pw := io.Pipe()

			// Write the request body as JSON.
			go func() {
				var err error
				defer func() {
					pw.CloseWithError(err)
				}()
				enc := json.NewEncoder(pw)
				reqHeader := batchPostRequestHeader{
					ID:           batchID,
					PaymentCount: len(payments),
				}
				err = enc.Encode(reqHeader)
				if err != nil {
					err = fmt.Errorf("encoding header: %w\n", err)
					return
				}
				for _, p := range payments {
					reqEntry := batchPostRequestEntry{
						Amount: p.Amount,
						Memo:   p.Memo,
					}
					err := enc.Encode(reqEntry)
					if err != nil {
						err = fmt.Errorf("encoding payment memo=%d: %w", p.Memo, err)
						return
					}
				}
			}()

			// Make the request using the request body streaming it to the destination.
			resp, err := http.Post(destinationAddr, "application/json", pr)
			if err != nil {
				return fmt.Errorf("sending batch meta to destination address: %w\n", err)
			}
			if resp.StatusCode/100 != 2 {
				respBody, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 512))
				return fmt.Errorf("sending batch meta to destination address: %s\n", string(respBody))
			}
			return nil
		})

		g.Go(func() error {
			seqNum, err := a.GetSequenceNumber()
			if err != nil {
				return fmt.Errorf("getting source account sequence number: %w\n", err)
			}
			tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
				SourceAccount: &txnbuild.SimpleAccount{
					AccountID: sourceAccountKey.Address(),
					Sequence:  seqNum + 1,
				},
				BaseFee:    txnbuild.MinBaseFee,
				Timebounds: txnbuild.NewInfiniteTimeout(),
				Memo:       txnbuild.MemoText(batchID),
				Operations: []txnbuild.Operation{
					&txnbuild.Payment{
						Destination: destinationAccount,
						Asset:       txnbuild.NativeAsset{},
						Amount:      amount.StringFromInt64(batchAmount),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("building tx: %w\n", err)
			}
			tx, err = tx.Sign(networkPassphrase, sourceAccountKey)
			if err != nil {
				return fmt.Errorf("signing tx: %w\n", err)
			}
			_, err = horizonClient.SubmitTransactionWithOptions(tx, horizonclient.SubmitTxOpts{SkipMemoRequiredCheck: true})
			if err != nil {
				return fmt.Errorf("submitting tx: %w\n", err)
			}
			return nil
		})

		err = g.Wait()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		s.AddPaymentsSent(int64(len(payments)))
		s.AddBatchesSent(1)
	}
}

type batchPostRequestHeader struct {
	ID           string
	PaymentCount int
}

type batchPostRequestEntry struct {
	Amount int64
	Memo   int64
}

type stats struct {
	timeStarted      time.Time
	timeFinished     time.Time
	paymentsSent     int64
	paymentsReceived int64
	batchesSent      int64
	batchesReceived  int64
}

func (s *stats) AddPaymentsSent(delta int64) {
	atomic.AddInt64(&s.paymentsSent, delta)
}

func (s *stats) AddPaymentsReceived(delta int64) {
	atomic.AddInt64(&s.paymentsReceived, delta)
}

func (s *stats) AddBatchesSent(delta int64) {
	atomic.AddInt64(&s.batchesSent, delta)
}

func (s *stats) AddBatchesReceived(delta int64) {
	atomic.AddInt64(&s.batchesReceived, delta)
}

func (s stats) String() string {
	timeSpent := s.timeFinished.Sub(s.timeStarted)
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "time spent: %v\n", timeSpent)
	fmt.Fprintf(&sb, "payments sent: %d\n", s.paymentsSent)
	fmt.Fprintf(&sb, "payments received: %d\n", s.paymentsReceived)
	fmt.Fprintf(&sb, "payments tps: %.3f\n", float64(s.paymentsSent+s.paymentsReceived)/timeSpent.Seconds())
	fmt.Fprintf(&sb, "batches sent: %d\n", s.batchesSent)
	fmt.Fprintf(&sb, "batches received: %d\n", s.batchesReceived)
	fmt.Fprintf(&sb, "batches tps: %.3f\n", float64(s.batchesSent+s.batchesReceived)/timeSpent.Seconds())
	return sb.String()
}

func newBatchID() string {
	uuid := uuid.New()
	return base64.StdEncoding.EncodeToString(uuid[:])
}
