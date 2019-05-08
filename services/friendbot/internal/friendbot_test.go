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
	mockSubmitTransaction := func(minion *Minion, channel chan SubmitResult, hclient *horizonclient.Client, txXDR string) {
		// Instead of submitting the tx, we emulate a success.
		txSuccess := hProtocol.TransactionSuccess{Env: txXDR}
		channel <- SubmitResult{
			maybeTransactionSuccess: &txSuccess,
			maybeErr:                nil,
		}
	}

	minionSeed := "SC6K74F25SXMNHFLXJDRIIJWBCEOWXDHU3R6RW43RZSZTA2XFIOFYCFT"
	minionKeypair, err := keypair.Parse(minionSeed)
	if !assert.NoError(t, err) {
		return
	}
	minions := []Minion{
		Minion{
			Account: Account{
				AccountID: minionKeypair.Address(),
				Sequence:  1,
			},
			Keypair: minionKeypair.(*keypair.Full),
		},
	}
	botSeed := "SC6K74F25SXMNHFLXJDRIIJWBCEOWXDHU3R6RW43RZSZTA2XFIOFYCFT"
	botKeypair, err := keypair.Parse(botSeed)
	if !assert.NoError(t, err) {
		return
	}
	fb := &Bot{
		Account:           Account{AccountID: botKeypair.Address()},
		Keypair:           botKeypair.(*keypair.Full),
		Network:           "Test SDF Network ; September 2015",
		StartingBalance:   "10000.00",
		SubmitTransaction: mockSubmitTransaction,
		Minions:           minions,
	}

	recipientAddress := "GDJIN6W6PLTPKLLM57UW65ZH4BITUXUMYQHIMAZFYXF45PZVAWDBI77Z"
	txSuccess, err := fb.Pay(recipientAddress)
	if !assert.NoError(t, err) {
		return
	}
	expectedTxn := "AAAAAGOiyZ/+kecCdxYBXywkAFwsSrGLYqD4IiVglvKCvaWHAAAAyAAAAAAAAAACAAAAAQAAAAAAAAAAAAAAAAAAASwAAAAAAAAAAgAAAAAAAAAAAAAAANKG+t565vUtbO/pb3cn4FE6XozEDoYDJcXLzr81BYYUAAAAF0h26AAAAAABAAAAAGOiyZ/+kecCdxYBXywkAFwsSrGLYqD4IiVglvKCvaWHAAAAAQAAAADShvreeub1LWzv6W93J+BROl6MxA6GAyXFy86/NQWGFAAAAAAAAAAXSHboAAAAAAAAAAACgr2lhwAAAEDcQhEvaKc/tNyDUWQtRYRH3MDZ/Aam3X/OPMbSWTozd/B2KzZzwEFj6qI5TpsDUFZ9OgYKJmYrsjOwQxxhdrMAgr2lhwAAAEDcQhEvaKc/tNyDUWQtRYRH3MDZ/Aam3X/OPMbSWTozd/B2KzZzwEFj6qI5TpsDUFZ9OgYKJmYrsjOwQxxhdrMA"
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
