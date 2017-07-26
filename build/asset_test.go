package build

import (
	"reflect"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsset_ToXDR(t *testing.T) {
	var (
		issuer        xdr.AccountId
		issuerAddress = "GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"
	)
	require.NoError(t, issuer.SetAddress(issuerAddress))

	type fields struct {
		Code   string
		Issuer string
		Native bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    xdr.Asset
		wantErr bool
	}{
		{
			name:   "Native",
			fields: fields{"", "", true},
			want:   xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
		},
		{
			name: "USD",
			fields: fields{
				"USD",
				issuerAddress,
				false,
			},
			want: xdr.Asset{
				Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
				AlphaNum4: &xdr.AssetAlphaNum4{
					AssetCode: [4]byte{0x55, 0x53, 0x44, 0x00}, //USD
					Issuer:    issuer,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Asset{
				Code:   tt.fields.Code,
				Issuer: tt.fields.Issuer,
				Native: tt.fields.Native,
			}
			got, err := a.ToXDR()
			if (err != nil) != tt.wantErr {
				t.Errorf("Asset.ToXDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Asset.ToXDR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsset_MustXDR(t *testing.T) {
	// good
	assert.NotPanics(t, func() {
		NativeAsset().MustXDR()
	})

	// bad
	assert.Panics(t, func() {
		CreditAsset("USD", "BONK").MustXDR()
	})
}
