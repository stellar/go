package internal

import (
	"log"

	hProtocol "github.com/stellar/go/protocols/horizon"
)

// Bot represents the friendbot subsystem and primarily delegates work
// to its Minions.
type Bot struct {
	Minions         []Minion
	nextMinionIndex int
}

// SubmitResult is the result from the asynchronous tx submission.
type SubmitResult struct {
	maybeTransactionSuccess *hProtocol.TransactionSuccess
	maybeErr                error
}

// Pay funds the account at `destAddress`.
func (bot *Bot) Pay(destAddress string) (*hProtocol.TransactionSuccess, error) {
	log.Printf("Selecting minion at index %d of max length %d", bot.nextMinionIndex, len(bot.Minions))
	minion := bot.Minions[bot.nextMinionIndex]
	resultChan := make(chan SubmitResult)
	go minion.Run(destAddress, resultChan)
	bot.nextMinionIndex = (bot.nextMinionIndex + 1) % len(bot.Minions)
	maybeSubmitResult := <-resultChan
	close(resultChan)
	return maybeSubmitResult.maybeTransactionSuccess, maybeSubmitResult.maybeErr
}
