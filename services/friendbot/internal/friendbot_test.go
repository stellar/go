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

	botSeed := "SC6K74F25SXMNHFLXJDRIIJWBCEOWXDHU3R6RW43RZSZTA2XFIOFYCFT"
	botKeypair, err := keypair.Parse(botSeed)
	if !assert.NoError(t, err) {
		return
	}
	botAccount := Account{AccountID: botKeypair.Address()}

	minionSeed := "SC6K74F25SXMNHFLXJDRIIJWBCEOWXDHU3R6RW43RZSZTA2XFIOFYCFT"
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
		StartingBalance:   "10000.00",
		InputChan:         make(chan MinionInput),
		SubmitTransaction: mockSubmitTransaction,
	}
	fb := &Bot{Minions: []Minion{minion}}
	go minion.Run()

	recipientAddress := "GDJIN6W6PLTPKLLM57UW65ZH4BITUXUMYQHIMAZFYXF45PZVAWDBI77Z"
	txSuccess, err := fb.Pay(recipientAddress)
	if !assert.NoError(t, err) {
		return
	}
	expectedTxn := "AAAAAGOiyZ/+kecCdxYBXywkAFwsSrGLYqD4IiVglvKCvaWHAAAAyAAAAAAAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAAAAAAANKG+t565vUtbO/pb3cn4FE6XozEDoYDJcXLzr81BYYUAAAAF0h26AAAAAABAAAAAGOiyZ/+kecCdxYBXywkAFwsSrGLYqD4IiVglvKCvaWHAAAAAQAAAADShvreeub1LWzv6W93J+BROl6MxA6GAyXFy86/NQWGFAAAAAAAAAAXSHboAAAAAAAAAAACgr2lhwAAAEBoQgfzvxb81HRrjMYgQbGwhhs4iXE+vqdLk9qayJJEc31HU6MMEmRCzusIJh7cpSdunqoqaxpYbXZVqtLAFd0Cgr2lhwAAAEBoQgfzvxb81HRrjMYgQbGwhhs4iXE+vqdLk9qayJJEc31HU6MMEmRCzusIJh7cpSdunqoqaxpYbXZVqtLAFd0C"
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
