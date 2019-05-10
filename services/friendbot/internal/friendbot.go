package internal

import (
	"log"

	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// Bot represents the friendbot subsystem and primarily delegates work
// to its Minions.
type Bot struct {
	Minions         []Minion
	nextMinionIndex int
}

// MinionInput is the input to the Minion from the Bot to construct a payment.
type MinionInput struct {
	destAddress string
	resultChan  chan SubmitResult
}

// SubmitResult is the result from the asynchronous tx submission.
type SubmitResult struct {
	maybeTransactionSuccess *hProtocol.TransactionSuccess
	maybeErr                error
}

// Pay funds the account at `destAddress`.
func (bot *Bot) Pay(destAddress string) (*hProtocol.TransactionSuccess, error) {
	log.Print("top of Pay")
	minion := bot.Minions[bot.nextMinionIndex]
	resultChan := make(chan SubmitResult)
	log.Printf("about to pass to minion %v", minion)
	minion.InputChan <- MinionInput{
		destAddress: destAddress,
		resultChan:  resultChan,
	}
	bot.nextMinionIndex = (bot.nextMinionIndex + 1) % len(bot.Minions)
	maybeSubmitResult := <-resultChan
	close(resultChan)
	// XXX: Remove debug nil check.
	log.Print("just got back potential result")
	if maybeSubmitResult.maybeErr != nil {
		switch e := maybeSubmitResult.maybeErr.(type) {
		case *horizonclient.Error:
			resCode, _ := e.ResultCodes()
			log.Print(resCode.TransactionCode)
			log.Print(resCode.OperationCodes)
		}
	}
	return maybeSubmitResult.maybeTransactionSuccess, maybeSubmitResult.maybeErr
}
