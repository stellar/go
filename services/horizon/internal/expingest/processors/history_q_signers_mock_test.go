package processors

import (
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stretchr/testify/mock"
)

type MockHistoryQSigners struct {
	mock.Mock
}

func (m *MockHistoryQSigners) AccountsForSigner(signer string, page db2.PageQuery) ([]history.AccountSigner, error) {
	a := m.Called(signer, page)
	return a.Get(0).([]history.AccountSigner), a.Error(1)
}

func (m *MockHistoryQSigners) CreateAccountSigner(account, signer string, weight int32) error {
	a := m.Called(account, signer, weight)
	return a.Error(0)
}

func (m *MockHistoryQSigners) RemoveAccountSigner(account, signer string) error {
	a := m.Called(account, signer)
	return a.Error(0)
}
