// Package config provides a common infrastructure for reading configuration
// data stored in local TOML files.
package config

import (
	"github.com/BurntSushi/toml"
	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
)

// InvalidConfigError is the error that is returned when an invalid
// configuration is encountered by the `Read` func.
type InvalidConfigError struct {
	InvalidFields map[string]string
}

// Read takes the TOML configuration file at `path`, parses it into `dest` and
// then uses github.com/asaskevich/govalidator to validate the struct.
func Read(path string, dest interface{}) error {
	_, err := toml.DecodeFile(path, dest)
	if err != nil {
		return errors.Wrap(err, "decode-file failed")
	}

	valid, err := govalidator.ValidateStruct(dest)

	if valid {
		return nil
	}

	fields := govalidator.ErrorsByField(err)

	return &InvalidConfigError{
		InvalidFields: fields,
	}
}

func init() {
	govalidator.SetFieldsRequiredByDefault(true)
}
