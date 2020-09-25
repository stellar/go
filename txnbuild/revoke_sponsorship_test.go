package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/xdr"
)

func TestRevokeSponsorship(t *testing.T) {
	accountAddress := "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"
	accountAddress2 := "GBUKBCG5VLRKAVYAIREJRUJHOKLIADZJOICRW43WVJCLES52BDOTCQZU"
	claimableBalanceId, err := xdr.MarshalHex(xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
	})
	assert.NoError(t, err)
	for _, testcase := range []struct {
		name string
		op   RevokeSponsorship
	}{
		{
			name: "Account",
			op: RevokeSponsorship{
				SponsorshipType: RevokeSponsorshipTypeAccount,
				Account:         &accountAddress,
			},
		},
		{
			name: "Account with source",
			op: RevokeSponsorship{
				SponsorshipType: RevokeSponsorshipTypeAccount,
				Account:         &accountAddress,
				SourceAccount: &SimpleAccount{
					AccountID: accountAddress2,
					// We intentionally don't set the sequence, since it isn't directly expressed in the XDR
					// Sequence:  1,
				},
			},
		},
		{
			name: "TrustLine",
			op: RevokeSponsorship{
				SponsorshipType: RevokeSponsorshipTypeTrustLine,
				TrustLine: &TrustLineId{
					Account: accountAddress,
					Asset: CreditAsset{
						Code:   "USD",
						Issuer: newKeypair0().Address(),
					},
				},
			},
		},
		{
			name: "Offer",
			op: RevokeSponsorship{
				SponsorshipType: RevokeSponsorshipTypeOffer,
				Offer: &OfferId{
					SellerAccountAddress: accountAddress,
					OfferId:              0xdeadbeef,
				},
			},
		},
		{
			name: "Data",
			op: RevokeSponsorship{
				SponsorshipType: RevokeSponsorshipTypeData,
				Data: &DataId{
					Account:  accountAddress,
					DataName: "foobar",
				},
			},
		},
		{
			name: "Data",
			op: RevokeSponsorship{
				SponsorshipType:  RevokeSponsorshipTypeClaimableBalance,
				ClaimableBalance: &claimableBalanceId,
			},
		},
		{
			name: "Signer",
			op: RevokeSponsorship{
				SponsorshipType: RevokeSponsorshipTypeSigner,
				Signer: &SignerId{
					AccountId:     accountAddress,
					SignerAddress: "XBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWGTOG",
				},
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			op := testcase.op
			assert.NoError(t, op.Validate())
			xdrOp, err := op.BuildXDR()
			assert.NoError(t, err)
			xdrBin, err := xdrOp.MarshalBinary()
			assert.NoError(t, err)
			var xdrOp2 xdr.Operation
			assert.NoError(t, xdr.SafeUnmarshal(xdrBin, &xdrOp2))
			var op2 RevokeSponsorship
			assert.NoError(t, op2.FromXDR(xdrOp2))
			assert.Equal(t, op, op2)
		})
	}
}
