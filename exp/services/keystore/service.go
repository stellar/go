package keystore

import (
	"context"
	"database/sql"

	"github.com/sirupsen/logrus"
)

type Config struct {
	DBURL          string
	MaxIdleDBConns int
	MaxOpenDBConns int

	LogFile  string
	LogLevel logrus.Level

	ListenerPort int
}

type Service struct {
	db *sql.DB
}

func NewService(ctx context.Context, db *sql.DB) *Service {
	return &Service{db: db}
}
