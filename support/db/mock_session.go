package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/mock"
)

var _ SessionInterface = (*MockSession)(nil)

type MockSession struct {
	mock.Mock
}

func (m *MockSession) Begin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSession) BeginTx(ctx context.Context, opts *sql.TxOptions) error {
	args := m.Called(ctx, opts)
	return args.Error(0)
}

func (m *MockSession) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSession) TruncateTables(ctx context.Context, tables []string) error {
	args := m.Called(ctx, tables)
	return args.Error(0)
}

func (m *MockSession) Clone() *Session {
	args := m.Called()
	return args.Get(0).(*Session)
}

func (m *MockSession) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSession) Get(ctx context.Context, dest interface{}, query sq.Sqlizer) error {
	args := m.Called(ctx, dest, query)
	return args.Error(0)
}

func (m *MockSession) GetRaw(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	argss := m.Called(ctx, dest, query, args)
	return argss.Error(0)
}

func (m *MockSession) Select(ctx context.Context, dest interface{}, query squirrel.Sqlizer) error {
	argss := m.Called(ctx, dest, query)
	return argss.Error(0)
}

func (m *MockSession) SelectRaw(ctx context.Context,
	dest interface{},
	query string,
	args ...interface{},
) error {
	argss := m.Called(ctx, dest, query, args)
	return argss.Error(0)
}

func (m *MockSession) GetTable(name string) *Table {
	args := m.Called(name)
	return args.Get(0).(*Table)
}

func (m *MockSession) Exec(ctx context.Context, query squirrel.Sqlizer) (sql.Result, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockSession) ExecRaw(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	argss := m.Called(ctx, query, args)
	return argss.Get(0).(sql.Result), argss.Error(1)
}

func (m *MockSession) NoRows(err error) bool {
	args := m.Called(err)
	return args.Get(0).(bool)
}

func (m *MockSession) Ping(ctx context.Context, timeout time.Duration) error {
	return m.Called(ctx, timeout).Error(0)
}
