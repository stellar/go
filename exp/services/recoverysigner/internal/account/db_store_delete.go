package account

func (s *DBStore) Delete(address string) error {
	// Delete an account will delete the associated identities and auth methods because of the ON DELETE CASCADE references.
	// https://github.com/stellar/go/blob/b3e0a353a901ce0babad5b4953330e55f2c674a1/exp/services/recoverysigner/internal/db/dbmigrate/migrations/20200311000001-create-identities.sql#L4
	// https://github.com/stellar/go/blob/b3e0a353a901ce0babad5b4953330e55f2c674a1/exp/services/recoverysigner/internal/db/dbmigrate/migrations/20200311000002-create-auth-methods.sql#L10
	result, err := s.DB.Exec(
		`DELETE FROM accounts
		WHERE address = $1`,
		address,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
