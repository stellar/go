package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	supporthttp "github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
	"github.com/stellar/go/support/render/httpjson"
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
	sourceAccount := "G..."
	listenAddr := ":8001"

	fs := flag.NewFlagSet("benchmark", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.BoolVar(&showHelp, "h", showHelp, "Show this help")
	fs.StringVar(&cpuProfileFile, "cpuprofile", cpuProfileFile, "Write cpu profile to `file`")
	fs.StringVar(&memProfileFile, "memprofile", memProfileFile, "Write memory profile to `file`")
	fs.StringVar(&horizonURL, "horizon", horizonURL, "Horizon URL")
	fs.StringVar(&sourceAccount, "source-account", sourceAccount, "Source account to receive payments at")
	fs.StringVar(&listenAddr, "listen", listenAddr, "Address and port to listen on")

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

	if sourceAccount == "G..." {
		sourceAccount = keypair.MustRandom().Address()
	}
	fmt.Printf("Source: %s\n", sourceAccount)

	fmt.Printf("Funding... %s\n", sourceAccount)
	_, err = horizonClient.Fund(sourceAccount)
	if err != nil {
		fmt.Printf("Error funding source: %v\n", err)
	}

	s := stats{}

	// Receive batch metadata via HTTP.
	logger := log.New()
	logger.Logger.Level = logrus.TraceLevel
	mux := supporthttp.NewAPIMux(logger)
	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Post("/batch", handleBatchFunc(&s))
	supporthttp.Run(supporthttp.Config{
		ListenAddr:  listenAddr,
		ReadTimeout: 120 * time.Second,
		Handler:     mux,
		OnStarting: func() {
			logger.Infof("Starting listening on %s...", listenAddr)
		},
		OnStopping: func() {
			logger.Info("Stopping...")
		},
		OnStopped: func() {
			logger.Info("Stopped")
			fmt.Println(s.String())
		},
	})

	return nil
}

func handleBatchFunc(s *stats) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx := r.Context()
		l := log.Ctx(ctx)

		byteCounter := NewCountReader(r.Body)
		z, err := gzip.NewReader(byteCounter)
		if err != nil {
			l.Warn(err.Error())
			httpjson.RenderStatus(w, http.StatusBadRequest, errorResponse{Error: err.Error()}, httpjson.JSON)
			return
		}
		dec := json.NewDecoder(z)

		reqHeader := batchPostRequestHeader{}
		err = dec.Decode(&reqHeader)
		if err != nil {
			l.Warn(err.Error())
			httpjson.RenderStatus(w, http.StatusBadRequest, errorResponse{Error: err.Error()}, httpjson.JSON)
			return
		}

		l = l.WithField("batch", reqHeader.ID).
			WithField("payments", reqHeader.PaymentCount)
		l.Info("Receiving batch.")

		count := 0
		for {
			reqEntry := batchPostRequestEntry{}
			err = dec.Decode(&reqEntry)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				l.Warn(err.Error())
				httpjson.RenderStatus(w, http.StatusBadRequest, errorResponse{Error: err.Error()}, httpjson.JSON)
				return
			}
			count++
			// In a real world application, do something with the entry.
		}

		l.Infof("Read %d bytes.", byteCounter.Count)

		// Validate the batch.
		if count != reqHeader.PaymentCount {
			msg := "incomplete batch received"
			l.Warn(msg)
			httpjson.RenderStatus(w, http.StatusBadRequest, errorResponse{Error: msg}, httpjson.JSON)
			return
		}
		// TODO: Validate the payment against payment on Horizon?

		l.Info("Received batch.")
		s.AddPaymentsReceived(int64(count))
		s.AddBatchesReceived(1)
		httpjson.RenderStatus(w, http.StatusAccepted, batchPostResponse{Result: "accepted"}, httpjson.JSON)
	}
}

type errorResponse struct {
	Error string
}

type batchPostRequestHeader struct {
	ID           string
	PaymentCount int
}

type batchPostRequestEntry struct {
	Amount int64
	Memo   int64
}

type batchPostResponse struct {
	Result string
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

func NewCountReader(r io.Reader) *CountReader {
	return &CountReader{Reader: r}
}

type CountReader struct {
	io.Reader
	Count int
}

func (r *CountReader) Read(b []byte) (n int, err error) {
	n, err = r.Reader.Read(b)
	r.Count += n
	return n, err
}
