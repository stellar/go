package internal

import (
	"log"
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stretchr/testify/assert"

	"sync"
)

func TestFriendbot_Pay(t *testing.T) {
	mockSubmitTransaction := func(minion *Minion, hclient *horizon.Client, signed string) {
		txSuccess := horizon.TransactionSuccess{Env: signed}
		// we don't want to submit the tx but emulate a success instead
		minion.TxResultChan <- TxResult{
			maybeTransactionSuccess: &txSuccess,
			maybeErr:                nil,
		}
	}

	minions := []Minion{
		Minion{
			Secret:       "SC6K74F25SXMNHFLXJDRIIJWBCEOWXDHU3R6RW43RZSZTA2XFIOFYCFT",
			DestAddrChan: make(chan string),
			TxResultChan: make(chan TxResult),
			sequence:     2,
		},
	}
	fb := &Bot{
		Secret:            "SAQWC7EPIYF3XGILYVJM4LVAVSLZKT27CTEI3AFBHU2VRCMQ3P3INPG5",
		Network:           "Test SDF Network ; September 2015",
		StartingBalance:   "100.00",
		SubmitTransaction: mockSubmitTransaction,
		Minions:           minions,
		nextMinionIndex:   0,
	}

	txSuccess, err := fb.Pay("GDJIN6W6PLTPKLLM57UW65ZH4BITUXUMYQHIMAZFYXF45PZVAWDBI77Z")
	if !assert.NoError(t, err) {
		return
	}

	log.Print(txSuccess.Env)

	expectedTxn := "AAAAAGOiyZ/+kecCdxYBXywkAFwsSrGLYqD4IiVglvKCvaWHAAAAyAAAAAAAAAADAAAAAAAAAAAAAAACAAAAAAAAAAAAAAAA0ob63nrm9S1s7+lvdyfgUTpejMQOhgMlxcvOvzUFhhQAAAAAAAAAAAAAAAEAAAAA+5h/vHsoa8Vf1+MJH1YhqhNffIcljBfplLHrDvoc+MQAAAABAAAAANKG+t565vUtbO/pb3cn4FE6XozEDoYDJcXLzr81BYYUAAAAAAAAAAA7msoAAAAAAAAAAAL6HPjEAAAAQK+pRYAmYSks2TwQI32M5f6l43HD19tr96xfMhTAzt8JBoycWrsqQd2wyBI43SIXAoJyqq/wi9xGf0WReDFF4AuCvaWHAAAAQF3Ipfu8bgH3JNewaJRMAZDNcb+gGLIHoM6+u7lsqWkhkmTlP51BK0CqG9BybkjoGQsObjtqqScmmy7g2pWR2AI="
	assert.Equal(t, expectedTxn, txSuccess.Env)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		_, err := fb.Pay("GDJIN6W6PLTPKLLM57UW65ZH4BITUXUMYQHIMAZFYXF45PZVAWDBI77Z")
		// don't assert on the txn value here because the ordering is not guaranteed between these 2 goroutines
		assert.NoError(t, err)
		wg.Done()
	}()
	go func() {
		_, err := fb.Pay("GDJIN6W6PLTPKLLM57UW65ZH4BITUXUMYQHIMAZFYXF45PZVAWDBI77Z")
		// don't assert on the txn value here because the ordering is not guaranteed between these 2 goroutines
		assert.NoError(t, err)
		wg.Done()
	}()
	wg.Wait()
}
