package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_db_type(t *testing.T) {
	c := Config{
		Port:              func(i int) *int { return &i }(8001),
		Horizon:           "https://example.com",
		NetworkPassphrase: "Test SDF Network ; September 2015",
		Database: &Database{
			Type: "",
			URL:  "",
		},
	}

	testCases := []struct {
		dbType  string
		wantErr error
	}{
		{dbType: "", wantErr: nil},
		{dbType: "postgres", wantErr: nil},
		{dbType: "mysql", wantErr: errors.New("Invalid database.type param, mysql support is discontinued")},
		{dbType: "bogus", wantErr: errors.New("Invalid database.type param")},
	}

	for _, tc := range testCases {
		t.Run(tc.dbType, func(t *testing.T) {
			c.Database.Type = tc.dbType
			err := c.Validate()
			if tc.wantErr == nil {
				assert.Nil(t, err)
			} else {
				require.NotNil(t, err)
				assert.Equal(t, tc.wantErr.Error(), err.Error())
			}
		})
	}
}

func TestConfig_Validate_db_url(t *testing.T) {
	c := Config{
		Port:              func(i int) *int { return &i }(8001),
		Horizon:           "https://example.com",
		NetworkPassphrase: "Test SDF Network ; September 2015",
		Database: &Database{
			Type: "",
			URL:  "",
		},
	}

	testCases := []struct {
		url     string
		wantErr error
	}{
		{url: "", wantErr: nil},
		{url: "postgres://localhost/db", wantErr: nil},
		{url: " postgres:", wantErr: errors.New("Cannot parse database.url param")},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			c.Database.URL = tc.url
			err := c.Validate()
			if tc.wantErr == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
			}
		})
	}
}
