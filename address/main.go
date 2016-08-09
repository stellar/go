// Package address provides utility functions for working with stellar
// addresses. See https://www.stellar.org/developers/guides/concepts/federation.
// html#stellar-addresses for more on addresses.
package address

import (
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/stellar/go/internal/errors"
)

// Separator seperates the name and domain portions of an address
const Separator = "*"

var (
	// ErrInvalidAddress is the error returned when an address is invalid in
	// such a way that we do not know if the name or domain portion is at fault.
	ErrInvalidAddress = errors.New("invalid address")

	// ErrInvalidName is the error returned when an address's name portion is
	// invalid.
	ErrInvalidName = errors.New("name part of address is invalid")

	// ErrInvalidDomain is the error returned when an address's domain portion
	// is invalid.
	ErrInvalidDomain = errors.New("domain part of address is invalid")
)

// Split takes an address, of the form "name*domain" and provides the
// constituent elements.
func Split(address string) (name, domain string, err error) {
	parts := strings.Split(address, Separator)

	if len(parts) != 2 {
		err = ErrInvalidAddress
		return
	}

	name = parts[0]
	domain = parts[1]

	if name == "" {
		err = ErrInvalidName
		return
	}

	if !govalidator.IsDNSName(domain) {
		err = ErrInvalidDomain
		return
	}

	return
}
