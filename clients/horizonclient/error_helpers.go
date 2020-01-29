package horizonclient

// IsNotFoundError returns true if the error is a horizonclient.Error with
// a not_found problem indicating that the resource is not found on
// Horizon.
func IsNotFoundError(err error) bool {
	var hErr *Error

	switch err := err.(type) {
	case *Error:
		hErr = err
	case Error:
		hErr = &err
	}

	if hErr == nil {
		return false
	}

	return hErr.Problem.Type == "https://stellar.org/horizon-errors/not_found"
}
