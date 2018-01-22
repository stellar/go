package database

import (
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

var errRollbackTrigger = errors.New("rollback trigger")

func WithTx(session *db.Session, f func(s *db.Session) error) error {
	newSession := session.Clone()
	if err := newSession.Begin(); err != nil {
		return errors.Wrap(err, "failed to start db transaction")
	}
	defer newSession.Rollback()

	if err := f(newSession); err != nil {
		// is reserved  error for flow control
		if err == errRollbackTrigger {
			return nil
		}
		return err
	}
	return newSession.Commit()
}

func ReadOnly(f func(s *db.Session) error) func(s *db.Session) error {
	return func(s *db.Session) error {
		if _, err := s.ExecRaw("SET transaction READ ONLY"); err != nil {
			return err
		}
		if err := f(s); err != nil {
			return err
		}
		return errRollbackTrigger
	}
}
