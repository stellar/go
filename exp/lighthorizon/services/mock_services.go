package services

import (
	"context"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stretchr/testify/mock"
)

type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) GetTransactionsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Transaction, error) {
	args := m.Called(ctx, cursor, limit, accountId)
	return args.Get(0).([]common.Transaction), args.Error(1)
}

type MockOperationService struct {
	mock.Mock
}

func (m *MockOperationService) GetOperationsByAccount(ctx context.Context,
	cursor int64, limit uint64,
	accountId string,
) ([]common.Operation, error) {
	args := m.Called(ctx, cursor, limit, accountId)
	return args.Get(0).([]common.Operation), args.Error(1)
}
