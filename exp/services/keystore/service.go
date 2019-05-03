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
}

type Service struct {
	db *sql.DB
}

func NewService(ctx context.Context, db *sql.DB) (*Service, error) {
	return &Service{db: db}, nil
}
