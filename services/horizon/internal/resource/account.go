package resource

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/render/hal"
)

// Populate fills out the resource's fields
func (this *Account) Populate(
	ctx context.Context,
	ca core.Account,
	cd []core.AccountData,
	cs []core.Signer,
	ct []core.Trustline,
	ha history.Account,
) (err error) {
	this.ID = ca.Accountid
	this.AccountID = ca.Accountid
	this.Sequence = ca.Seqnum
	this.SubentryCount = ca.Numsubentries
	this.InflationDestination = ca.Inflationdest.String
	this.HomeDomain = ca.HomeDomain.String

	this.Flags.Populate(ca)
	this.Thresholds.Populate(ca)

	// populate balances
	this.Balances = make([]Balance, len(ct)+1)
	for i, tl := range ct {
		err = this.Balances[i].Populate(ctx, tl)
		if err != nil {
			return
		}
	}

	// add native balance
	err = this.Balances[len(this.Balances)-1].PopulateNative(ca.Balance)
	if err != nil {
		return
	}

	// populate data
	this.Data = make(map[string]string)
	for _, d := range cd {
		this.Data[d.Key] = d.Value
	}

	// populate signers
	this.Signers = make([]Signer, len(cs)+1)
	for i, s := range cs {
		this.Signers[i].Populate(ctx, s)
	}

	this.Signers[len(this.Signers)-1].PopulateMaster(ca)

	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	self := fmt.Sprintf("/accounts/%s", ca.Accountid)
	this.Links.Self = lb.Link(self)
	this.Links.Transactions = lb.PagedLink(self, "transactions")
	this.Links.Operations = lb.PagedLink(self, "operations")
	this.Links.Payments = lb.PagedLink(self, "payments")
	this.Links.Effects = lb.PagedLink(self, "effects")
	this.Links.Offers = lb.PagedLink(self, "offers")
	this.Links.Trades = lb.PagedLink(self, "trades")
	this.Links.Data = lb.Link(self, "data/{key}")
	this.Links.Data.PopulateTemplated()
	return
}

// MustGetData returns decoded value for a given key. If the key does
// not exist, empty slice will be returned. If there is an error
// decoding a value, it will panic.
func (this *Account) MustGetData(key string) []byte {
	bytes, err := this.GetData(key)
	if err != nil {
		panic(err)
	}
	return bytes
}

// GetData returns decoded value for a given key. If the key does
// not exist, empty slice will be returned.
func (this *Account) GetData(key string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(this.Data[key])
}
