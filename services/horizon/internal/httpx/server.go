package httpx

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"

	"github.com/stellar/go/services/horizon/internal/db2"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
)

type ServerMetrics struct {
	RequestTimer metrics.Timer
	FailureMeter metrics.Meter
	SuccessMeter metrics.Meter
}

func NewServerMetrics(registry metrics.Registry) *ServerMetrics {
	result := ServerMetrics{
		RequestTimer: metrics.NewTimer(),
		FailureMeter: metrics.NewMeter(),
		SuccessMeter: metrics.NewMeter(),
	}
	registry.Register("requests.total", result.RequestTimer)
	registry.Register("requests.succeeded", result.SuccessMeter)
	registry.Register("requests.failed", result.FailureMeter)
	return &result
}

// Web contains the http server related fields for horizon: the Router,
// rate limiter, etc.
type Server struct {
	Router         *Router
	server         *http.Server
	internalServer *http.Server
}

func init() {
	// register problems
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)
	problem.RegisterError(sequence.ErrNoMoreRoom, hProblem.ServerOverCapacity)
	problem.RegisterError(db2.ErrInvalidCursor, problem.BadRequest)
	problem.RegisterError(db2.ErrInvalidLimit, problem.BadRequest)
	problem.RegisterError(db2.ErrInvalidOrder, problem.BadRequest)
	problem.RegisterError(sse.ErrRateLimited, hProblem.RateLimitExceeded)
	problem.RegisterError(context.DeadlineExceeded, hProblem.Timeout)
	problem.RegisterError(context.Canceled, hProblem.ServiceUnavailable)
	problem.RegisterError(db.ErrCancelled, hProblem.ServiceUnavailable)
}

func NewServer(config *RouterConfig) (*Server, error) {
	sm := NewServerMetrics(config.MetricsRegistry)
	router, err := NewRouter(config, sm)
	if err != nil {
		return nil, err
	}
	result := &Server{
		Router: router,
	}
	return result, nil
}
func (s *Server) Serve(port uint16, certFile, keyFile string, adminPort uint16) error {
	if s.server != nil {
		return errors.New("server already started")
	}

	if adminPort != 0 {
		go func() {
			adminAddr := fmt.Sprintf(":%d", adminPort)
			log.Infof("Starting internal server on %s", adminAddr)
			s.internalServer = &http.Server{
				Addr:        adminAddr,
				Handler:     s.Router.Internal,
				ReadTimeout: 5 * time.Second,
			}
			if err := s.internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Warn(errors.Wrap(err, "error in internalServer.ListenAndServe()"))
			}
		}()
	}

	addr := fmt.Sprintf(":%d", port)
	s.server = &http.Server{
		Addr:        addr,
		Handler:     s.Router,
		ReadTimeout: 5 * time.Second,
	}

	var err error
	if certFile != "" {
		err = s.server.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = s.server.ListenAndServe()
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	if s.internalServer != nil {
		wg.Add(1)
		go func() {
			err := s.internalServer.Shutdown(ctx)
			if err != nil {
				log.Warn(errors.Wrap(err, "error in internalServer.Shutdown()"))
			}
			wg.Done()
		}()
	}
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
