package db

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type roundTripProbe struct {
	session              SessionInterface
	roundTripTimeSummary prometheus.Summary

	closeChan chan struct{}
	closeOnce sync.Once
}

func (p *roundTripProbe) start() {
	p.closeChan = make(chan struct{})
	// session must be cloned because will be used concurrently in a
	// separate go routine in roundTripProbe
	p.session = p.session.Clone()

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				startTime := time.Now()
				_, err := p.session.ExecRaw(ctx, "select 1")
				if err == nil {
					p.roundTripTimeSummary.Observe(time.Since(startTime).Seconds())
				}
				cancel()
			case <-p.closeChan:
				return
			}
		}
	}()
}

func (p *roundTripProbe) close() {
	p.closeOnce.Do(func() {
		close(p.closeChan)
	})
}
