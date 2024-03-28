package stellarcore

import (
	"context"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockClientWithMetrics struct {
	mock.Mock
}

// SubmitTransaction mocks the SubmitTransaction method
func (m *MockClientWithMetrics) SubmitTransaction(ctx context.Context, rawTx string, envelope xdr.TransactionEnvelope) (*proto.TXResponse, error) {
	args := m.Called(ctx, rawTx, envelope)
	return args.Get(0).(*proto.TXResponse), args.Error(1)
}

func (m *MockClientWithMetrics) UpdateTxSubMetrics(duration float64, envelope xdr.TransactionEnvelope, response *proto.TXResponse, err error) {
	m.Called(duration, envelope, response, err)
}
