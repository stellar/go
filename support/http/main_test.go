package http

import (
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun_setup(t *testing.T) {

	// test that using no handler panics
	assert.Panics(t, func() {
		setup(Config{
			Handler: nil,
		})
	})

	// test defaults
	srv := setup(Config{
		Handler: stdhttp.NotFoundHandler(),
	})

	assert.Equal(t, defaultShutdownGracePeriod, srv.Timeout)
	assert.Equal(t, defaultReadTimeout, srv.ReadTimeout)
	assert.Equal(t, time.Duration(0), srv.WriteTimeout)
	assert.Equal(t, time.Duration(0), srv.IdleTimeout)
	assert.Equal(t, defaultListenAddr, srv.Server.Addr)
}
