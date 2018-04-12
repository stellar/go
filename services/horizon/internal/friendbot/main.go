package friendbot

import (
	"context"
	"errors"
	"sync"

	. "github.com/stellar/go/build"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/txsub"
)

// Bot represents the friendbot subsystem.
type Bot struct {
	Submitter *txsub.System
	Secret    string
	Network   string

	sequence uint64
	lock     sync.Mutex
}

// Pay funds the account at `address`
func (bot *Bot) Pay(ctx context.Context, address string) (result txsub.Result) {

	// establish initial sequence if needed
	if bot.sequence == 0 {
		result.Err = bot.refreshSequence(ctx)
		if result.Err != nil {
			return
		}
	}

	var envelope string
	envelope, result.Err = bot.makeTx(address)
	if result.Err != nil {
		return
	}

	resultChan := bot.Submitter.Submit(ctx, envelope)

	select {
	case result := <-resultChan:
		if result.Err != nil {
			bot.refreshSequence(ctx)
		}
		return result
	case <-ctx.Done():
		return txsub.Result{Err: txsub.ErrCanceled}
	}
}

func (bot *Bot) makeTx(address string) (string, error) {
	bot.lock.Lock()
	defer bot.lock.Unlock()

	tx, err := Transaction(
		SourceAccount{bot.Secret},
		Sequence{bot.sequence + 1},
		Network{bot.Network},
		CreateAccount(
			Destination{address},
			NativeAmount{"10000.00"},
		),
	)

	if err != nil {
		return "", err
	}

	bot.sequence++

	txe, err := tx.Sign(bot.Secret)
	if err != nil {
		return "", err
	}

	return txe.Base64()
}

func (bot *Bot) refreshSequence(ctx context.Context) error {
	bot.lock.Lock()
	defer bot.lock.Unlock()

	addy := bot.address()
	sp := bot.Submitter.Sequences

	seqs, err := sp.Get([]string{addy})
	if err != nil {
		return err
	}

	seq, ok := seqs[addy]
	if !ok {
		return errors.New("friendbot account not found")
	}

	bot.sequence = seq
	return nil
}

func (bot *Bot) address() string {
	kp := keypair.MustParse(bot.Secret)
	return kp.Address()
}
