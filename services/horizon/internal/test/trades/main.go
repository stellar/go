//Common infrastructure for testing Trades
package trades

import (
	. "github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

//getTestAsset generates an issuer on the fly and creates a CreditAlphanum4 Asset with given code
func getTestAsset(code string, accountNumber byte) xdr.Asset {
	var codeBytes [4]byte
	copy(codeBytes[:], []byte(code))
	ca4 := xdr.AssetAlphaNum4{Issuer: getTestAccount(accountNumber), AssetCode: codeBytes}
	return xdr.Asset{Type: xdr.AssetTypeAssetTypeCreditAlphanum4, AlphaNum4: &ca4, AlphaNum12: nil}
}

//Get generates and returns an account on the fly
func getTestAccount(accountNumber byte) xdr.AccountId {
	accountNumber++
	acc, _ := xdr.NewAccountId(xdr.PublicKeyTypePublicKeyTypeEd25519, xdr.Uint256{accountNumber})
	return acc
}

//ingestTestTrade mock ingests a trade
func ingestTestTrade(
	q *Q,
	assetSold xdr.Asset,
	assetBought xdr.Asset,
	seller xdr.AccountId,
	buyer xdr.AccountId,
	amountSold int64,
	amountBought int64,
	timestamp int64,
	opCounter int64) error {

	trade := xdr.ClaimOfferAtom{}
	trade.AmountBought = xdr.Int64(amountBought)
	trade.SellerId = seller
	trade.AmountSold = xdr.Int64(amountSold)
	trade.AssetBought = assetBought
	trade.AssetSold = assetSold

	opCounter++
	return q.InsertTrade(opCounter, 0, buyer, trade, timestamp)
}

//PopulateTestTrades generates and ingests trades between two assets according to given parameters
func PopulateTestTrades(q *Q, startTs int64, numOfTrades int, delta int64) (err error, ass1 xdr.Asset, ass2 xdr.Asset) {
	acc1 := getTestAccount(1)
	acc2 := getTestAccount(2)
	ass1 = getTestAsset("usd", 3)
	ass2 = getTestAsset("euro", 4)

	for i := 1; i <= numOfTrades; i++ {
		err = ingestTestTrade(
			q, ass1, ass2, acc1, acc2, int64(i*100), int64(i*100)*int64(i), startTs+(delta*int64(i-1)),int64(i))
		//tt.Assert.NoError(err)
		if err != nil {
			return
		}
	}
	return
}
