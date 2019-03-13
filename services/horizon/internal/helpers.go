package horizon

import (
	"github.com/stellar/go/support/errors"
)

func countNonEmpty(params ...interface{}) (int, error) {
	count := 0

	for _, param := range params {
		switch param := param.(type) {
		default:
			return 0, errors.Errorf("unexpected type %T", param)
		case int32:
			if param != 0 {
				count++
			}
		case string:
			if param != "" {
				count++
			}
		}
	}

	return count, nil
}
