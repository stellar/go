package keystore

import (
	"context"
	"database/sql"

	"github.com/sirupsen/logrus"
)

const (
	REST    = "REST"
	GraphQL = "GRAPHQL"
)

type Config struct {
	DBURL          string
	MaxIdleDBConns int
	MaxOpenDBConns int

	LogFile  string
	LogLevel logrus.Level

	AUTHURL string

	ListenerPort int
}

type Authenticator struct {
	URL     string
	APIType string
	//GraphQL related fields will be added later
}

type Service struct {
	db            *sql.DB
	authenticator *Authenticator
}

func NewService(ctx context.Context, db *sql.DB, authenticator *Authenticator) *Service {
	return &Service{db: db, authenticator: authenticator}
}
