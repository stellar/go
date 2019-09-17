package actions

import (
	"testing"

	"github.com/asaskevich/govalidator"
	"github.com/stretchr/testify/assert"
)

func TestAssetTypeValidator(t *testing.T) {
	type Query struct {
		AssetType string `valid:"assetType,optional"`
	}

	for _, testCase := range []struct {
		assetType string
		valid     bool
	}{
		{
			"native",
			true,
		},
		{
			"credit_alphanum4",
			true,
		},
		{
			"credit_alphanum12",
			true,
		},
		{
			"",
			true,
		},
		{
			"stellar_asset_type",
			false,
		},
	} {
		t.Run(testCase.assetType, func(t *testing.T) {
			tt := assert.New(t)

			q := Query{
				AssetType: testCase.assetType,
			}

			result, err := govalidator.ValidateStruct(q)
			if testCase.valid {
				tt.NoError(err)
				tt.True(result)
			} else {
				tt.Equal("AssetType: stellar_asset_type does not validate as assetType", err.Error())
			}
		})
	}
}

func TestAccountIDValidator(t *testing.T) {
	type Query struct {
		Account string `valid:"accountID,optional"`
	}

	for _, testCase := range []struct {
		name          string
		value         string
		expectedError string
	}{
		{
			"invalid stellar address",
			"FON4WOTCFSASG3J6SGLLQZURDDUVNBQANAHEQJ3PBNDZ74X63UZWQPZW",
			"Account: FON4WOTCFSASG3J6SGLLQZURDDUVNBQANAHEQJ3PBNDZ74X63UZWQPZW does not validate as accountID",
		},
		{
			"valid stellar address",
			"GAN4WOTCFSASG3J6SGLLQZURDDUVNBQANAHEQJ3PBNDZ74X63UZWQPZW",
			"",
		},
		{
			"empty stellar address should not be validated",
			"",
			"",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tt := assert.New(t)

			q := Query{
				Account: testCase.value,
			}

			result, err := govalidator.ValidateStruct(q)
			if testCase.expectedError == "" {
				tt.NoError(err)
				tt.True(result)
			} else {
				tt.Equal(testCase.expectedError, err.Error())
			}
		})
	}
}
