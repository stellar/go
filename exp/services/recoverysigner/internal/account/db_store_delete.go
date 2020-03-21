package account

func (s *DBStore) Delete(address string) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}

	result, err := tx.Exec(
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

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil

	return nil
}
