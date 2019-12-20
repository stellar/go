package main

import (
	"github.com/sirupsen/logrus"
	"github.com/stellar/go/exp/tools/jwtkeygen/commands"
	supportlog "github.com/stellar/go/support/log"
)

type plainFormatter struct{}

func (plainFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

func main() {
	logger := supportlog.New()
	logger.Logger.Level = logrus.TraceLevel
	logger.Logger.Formatter = &plainFormatter{}

	rootCmd := (&commands.GenJWTKeyCommand{Logger: logger}).Command()

	err := rootCmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
