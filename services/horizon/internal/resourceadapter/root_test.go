package resourceadapter

import (
	"context"
	"net/url"
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/horizon/internal/ledger"
)

func TestPopulateRoot(t *testing.T) {
	res := &horizon.Root{}
	PopulateRoot(context.Background(),
		res,
		ledger.State{CoreLatest: 1, HistoryLatest: 3, HistoryElder: 2},
		"hVersion",
		"cVersion",
		"passphrase",
		100,
		urlMustParse(t, "https://friendbot.example.com"))
	if res.CoreSequence != 1 {
		t.Errorf("CoreSequence did not match expectation %d", res.CoreSequence)
	}
	if res.HistoryElderSequence != 2 {
		t.Errorf("HistoryElderSequence did not match expectation %d", res.HistoryElderSequence)
	}
	if res.HorizonSequence != 3 {
		t.Errorf("HorizonSequence did not match expectation %d", res.HorizonSequence)
	}
	if res.StellarCoreVersion != "cVersion" {
		t.Errorf("StellarCoreVersion did not match expectation %s", res.StellarCoreVersion)
	}
	if res.HorizonVersion != "hVersion" {
		t.Errorf("HorizonVersion did not match expectation %s", res.HorizonVersion)
	}
	if res.NetworkPassphrase != "passphrase" {
		t.Errorf("Network passphrase did not match expectation %s", res.NetworkPassphrase)
	}
	if res.Links.Friendbot.Href != "https://friendbot.example.com/{?addr}" {
		t.Errorf("Friendbot URL not set as expected %s", res.Links.Friendbot.Href)
	}
}

func urlMustParse(t *testing.T, s string) *url.URL {
	if u, err := url.Parse(s); err != nil {
		t.Fatalf("Unable to parse URL: %s/%v", s, err)
		return nil
	} else {
		return u
	}
}
