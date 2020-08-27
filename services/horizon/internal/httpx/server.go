package httpx

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/services/horizon/internal/db2"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
)

type ServerMetrics struct {
	RequestDurationSummary *prometheus.SummaryVec
}

// Web contains the http server related fields for horizon: the Router,
// rate limiter, etc.
type Server struct {
	Router   *Router
	Metrics  *ServerMetrics
	server   *http.Server
	tlsFiles *struct {
		certFile, keyFile string
	}
	internalServer *http.Server
	sync.RWMutex
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

func NewServer(config *RouterConfig, port uint16, certFile, keyFile string, adminPort uint16) (*Server, error) {
	sm := &ServerMetrics{
		RequestDurationSummary: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: "horizon", Subsystem: "http", Name: "requests_duration_seconds",
				Help: "HTTP requests durations, sliding window = 10m",
			},
			[]string{"status", "route", "streaming", "method"},
		),
	}
	router, err := NewRouter(config, sm)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf(":%d", port)
	result := &Server{
		Router:  router,
		Metrics: sm,
		server: &http.Server{
			Addr:        addr,
			Handler:     router,
			ReadTimeout: 5 * time.Second,
		},
	}
	if certFile != "" && keyFile != "" {
		result.tlsFiles = &struct {
			certFile, keyFile string
		}{keyFile, certFile}
	}
	if adminPort != 0 {
		adminAddr := fmt.Sprintf(":%d", adminPort)
		result.internalServer = &http.Server{
			Addr:        adminAddr,
			Handler:     result.Router.Internal,
			ReadTimeout: 5 * time.Second,
		}
	}
	return result, nil
}
func (s *Server) Serve() error {
	if s.internalServer != nil {
		go func() {
			log.Infof("Starting internal server on %s", s.internalServer.Addr)
			if err := s.internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Warn(errors.Wrap(err, "error in internalServer.ListenAndServe()"))
			}
		}()
	}

	var err error
	if s.tlsFiles != nil {
		err = s.server.ListenAndServeTLS(s.tlsFiles.certFile, s.tlsFiles.keyFile)
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
