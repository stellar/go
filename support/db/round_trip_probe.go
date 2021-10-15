package db

import (
	"context"
	"sync"
	"time"
)

type roundTripProbe struct {
	session *SessionWithMetrics

	closeChan chan struct{}
	closeOnce sync.Once
}

func (p *roundTripProbe) start() {
	p.closeChan = make(chan struct{})

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				startTime := time.Now()
				_, err := p.session.ExecRaw(ctx, "select 1")
				if err == nil {
					p.session.roundTripTimeSummary.Observe(time.Since(startTime).Seconds())
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
