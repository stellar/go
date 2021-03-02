package horizon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var _ stellarCoreClient = (*mockStellarCore)(nil)

type mockStellarCore struct {
	mock.Mock
}

func (m *mockStellarCore) Info(ctx context.Context) (*stellarcore.InfoResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*stellarcore.InfoResponse), args.Error(1)
}

func TestHealthCheck(t *testing.T) {
	synced := &stellarcore.InfoResponse{}
	synced.Info.State = "Synced!"
	notSynced := &stellarcore.InfoResponse{}
	notSynced.Info.State = "Catching up"

	for _, tc := range []struct {
		name             string
		pingErr          error
		coreErr          error
		coreResponse     *stellarcore.InfoResponse
		expectedStatus   int
		expectedResponse healthResponse
	}{
		{
			"healthy",
			nil,
			nil,
			synced,
			http.StatusOK,
			healthResponse{
				DatabaseConnected: true,
				CoreUp:            true,
				CoreSynced:        true,
			},
		},
		{
			"db down",
			fmt.Errorf("database is down"),
			nil,
			synced,
			http.StatusServiceUnavailable,
			healthResponse{
				DatabaseConnected: false,
				CoreUp:            true,
				CoreSynced:        true,
			},
		},
		{
			"stellar core not synced",
			nil,
			nil,
			notSynced,
			http.StatusServiceUnavailable,
			healthResponse{
				DatabaseConnected: true,
				CoreUp:            true,
				CoreSynced:        false,
			},
		},
		{
			"stellar core down",
			nil,
			fmt.Errorf("stellar core is down"),
			nil,
			http.StatusServiceUnavailable,
			healthResponse{
				DatabaseConnected: true,
				CoreUp:            false,
				CoreSynced:        false,
			},
		},
		{
			"stellar core and db down",
			fmt.Errorf("database is down"),
			fmt.Errorf("stellar core is down"),
			nil,
			http.StatusServiceUnavailable,
			healthResponse{
				DatabaseConnected: false,
				CoreUp:            false,
				CoreSynced:        false,
			},
		},
		{
			"stellar core not synced and db down",
			fmt.Errorf("database is down"),
			nil,
			notSynced,
			http.StatusServiceUnavailable,
			healthResponse{
				DatabaseConnected: false,
				CoreUp:            true,
				CoreSynced:        false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := &db.MockSession{}
			session.On("Ping").Return(tc.pingErr).Once()
			ctx := context.Background()
			core := &mockStellarCore{}
			core.On("Info", ctx).Return(tc.coreResponse, tc.coreErr).Once()

			h := healthCheck{
				session: session,
				ctx:     ctx,
				core:    core,
			}

			w := httptest.NewRecorder()
			h.ServeHTTP(w, nil)
			assert.Equal(t, tc.expectedStatus, w.Code)

			var response healthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResponse, response)
		})
	}
}
