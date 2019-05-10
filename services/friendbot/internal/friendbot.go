package internal

import (
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
	minion := bot.Minions[bot.nextMinionIndex]
	resultChan := make(chan SubmitResult)
	minion.InputChan <- MinionInput{
		destAddress: destAddress,
		resultChan:  resultChan,
	}
	bot.nextMinionIndex = (bot.nextMinionIndex + 1) % len(bot.Minions)
	maybeSubmitResult := <-resultChan
	return maybeSubmitResult.maybeTransactionSuccess, maybeSubmitResult.maybeErr
}
