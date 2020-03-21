package account

import (
	"database/sql"
)

func (s *DBStore) Delete(address string) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}

	deletedAt := s.Clock.Now().UTC()

	accountID := int64(0)
	err = tx.Get(
		&accountID,
		`UPDATE accounts
		SET deleted_at = $1
		WHERE address = $2
		AND deleted_at IS NULL
		RETURNING id`,
		deletedAt,
		address,
	)
	if err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	_, err = tx.Exec(
		`UPDATE identities
		SET deleted_at = $1
		WHERE account_id = $2`,
		deletedAt,
		accountID,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`UPDATE auth_methods
		SET deleted_at = $1
		WHERE account_id = $2`,
		deletedAt,
		accountID,
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil

	return nil
}
