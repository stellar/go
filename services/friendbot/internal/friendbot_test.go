package internal

import (
	"sync"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stretchr/testify/assert"
)

func TestFriendbot_Pay(t *testing.T) {
	mockSubmitTransaction := func(minion *Minion, hclient *horizonclient.Client, tx string) (*hProtocol.TransactionSuccess, error) {
		// Instead of submitting the tx, we emulate a success.
		txSuccess := hProtocol.TransactionSuccess{Env: tx}
		return &txSuccess, nil
	}

	// Public key: GD25B4QI6KWVDWXDW25CIM7EKR6A6PBSWE2RCNSAC4NJQDQJXZJYMMKR
	botSeed := "SCWNLYELENPBXN46FHYXETT5LJCYBZD5VUQQVW4KZPHFO2YTQJUWT4D5"
	botKeypair, err := keypair.Parse(botSeed)
	if !assert.NoError(t, err) {
		return
	}
	botAccount := Account{AccountID: botKeypair.Address()}

	// Public key: GD4AGPPDFFHKK3Z2X4XZDRXX6GZQKP4FMLVQ5T55NDEYGG3GIP7BQUHM
	minionSeed := "SDTNSEERJPJFUE2LSDNYBFHYGVTPIWY7TU2IOJZQQGLWO2THTGB7NU5A"
	minionKeypair, err := keypair.Parse(minionSeed)
	if !assert.NoError(t, err) {
		return
	}

	minion := Minion{
		Account: Account{
			AccountID: minionKeypair.Address(),
			Sequence:  1,
		},
		Keypair:           minionKeypair.(*keypair.Full),
		BotAccount:        botAccount,
		BotKeypair:        botKeypair.(*keypair.Full),
		Network:           "Test SDF Network ; September 2015",
		StartingBalance:   "9999.00",
		SubmitTransaction: mockSubmitTransaction,
	}
	fb := &Bot{Minions: []Minion{minion}}

	recipientAddress := "GDJIN6W6PLTPKLLM57UW65ZH4BITUXUMYQHIMAZFYXF45PZVAWDBI77Z"
	txSuccess, err := fb.Pay(recipientAddress)
	if !assert.NoError(t, err) {
		return
	}
	expectedTxn := "AAAAAPgDPeMpTqVvOr8vkcb38bMFP4Vi6w7PvWjJgxtmQ/4YAAAAyAAAAAAAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAEAAAAA+AM94ylOpW86vy+RxvfxswU/hWLrDs+9aMmDG2ZD/hgAAAAAAAAAANKG+t565vUtbO/pb3cn4FE6XozEDoYDJcXLzr81BYYUAAAAAACYloAAAAABAAAAAPXQ8gjyrVHa47a6JDPkVHwPPDKxNRE2QBcamA4JvlOGAAAAAQAAAADShvreeub1LWzv6W93J+BROl6MxA6GAyXFy86/NQWGFAAAAAAAAAAXR95RgAAAAAAAAAACZkP+GAAAAEDpGRrZjQhnRxJwFoOA7jj2CPeYbGZcYAMke1I5XEb0qzXPgOfTyLlSolcrSzPT1sMvED7gEGc8s+bNiAe/5GYGCb5ThgAAAECd46/tB7j4h/5rUhmDobZT1DO0zK3DguZEyHQr1H4Wl7w2ozU5oP9oaWYH87H2mRnT1EfkUKnIFdnVBl6UAmAO"
	assert.Equal(t, expectedTxn, txSuccess.Env)

	// Don't assert on tx values below, since the completion order is unknown.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		_, err := fb.Pay(recipientAddress)
		assert.NoError(t, err)
		wg.Done()
	}()
	go func() {
		_, err := fb.Pay(recipientAddress)
		assert.NoError(t, err)
		wg.Done()
	}()
	wg.Wait()
}
