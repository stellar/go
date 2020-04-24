package serve

import (
	"net/http"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
)

// ChallengeHandler implements the SEP-10 challenge endpoint and handles
// requests for a new challenge transaction.
type challengeHandler struct {
	Logger             *supportlog.Entry
	ServerName         string
	NetworkPassphrase  string
	SigningKey         *keypair.Full
	ChallengeExpiresIn time.Duration
}

type challengeResponse struct {
	Transaction       string `json:"transaction"`
	NetworkPassphrase string `json:"network_passphrase"`
}

func (h challengeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	account := r.URL.Query().Get("account")
	if !strkey.IsValidEd25519PublicKey(account) {
		badRequest.Render(w)
		return
	}

	tx, err := txnbuild.BuildChallengeTx(
		h.SigningKey.Seed(),
		account,
		h.ServerName,
		h.NetworkPassphrase,
		h.ChallengeExpiresIn,
	)
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	hash, err := tx.HashHex()
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	l := h.Logger.Ctx(ctx).
		WithField("hash", hash).
		WithField("account", account)

	l.Infof("Generated challenge transaction %s for account %s.", hash, account)

	res := challengeResponse{
		Transaction:       tx,
		NetworkPassphrase: h.NetworkPassphrase,
	}
	httpjson.Render(w, res, httpjson.JSON)
}
