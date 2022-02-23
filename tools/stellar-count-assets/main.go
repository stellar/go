package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon/effects"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	client := horizonclient.Client{
		HorizonURL: "https://horizon.stellar.org",
	}

	w := csv.NewWriter(os.Stdout)
	err := w.Write([]string{"ledger", "asset count"})
	if err != nil {
		return fmt.Errorf("writing to csv: %w", err)
	}

	ledgers, err := client.Ledgers(horizonclient.LedgerRequest{
		Order: horizonclient.OrderDesc,
		Limit: 1,
	})
	if err != nil {
		return fmt.Errorf("retrieving last ledger: %w", err)
	}
	lastLedger := ledgers.Embedded.Records[0].Sequence

	// Sample 200 ledgers across a one week period.
	const ledgersInPeriod = 7 * (24 * 60 * 60) / 5
	const ledgersToSample = 200
	for i := int32(0); i < ledgersToSample; i++ {
		assets := map[string]int{}

		lid := lastLedger - (i * ledgersInPeriod / ledgersToSample)
		eff := effects.EffectsPage{}
		lastEff := effects.EffectsPage{}
		for {
			if eff.Links.Self.Href == "" {
				eff, err = client.Effects(horizonclient.EffectRequest{
					ForLedger: strconv.Itoa(int(lid)),
					Limit:     200,
				})
			} else {
				eff, err = client.NextEffectsPage(eff)
			}
			if err != nil {
				return fmt.Errorf("retrieving effects for ledger %q: %w", lid, err)
			}
			if eff.Links.Self == lastEff.Links.Self {
				break
			}

			for _, e := range eff.Embedded.Records {
				switch v := e.(type) {
				case effects.Trade:
					assets[assetStr(v.SoldAssetType, v.SoldAssetCode, v.SoldAssetIssuer)]++
					assets[assetStr(v.BoughtAssetType, v.BoughtAssetCode, v.BoughtAssetIssuer)]++
				case effects.LiquidityPoolTrade:
					assets[v.Sold.Asset]++
					assets[v.Bought.Asset]++
				}
			}

			lastEff = eff
		}

		err = w.Write([]string{
			strconv.Itoa(int(lid)),
			strconv.Itoa(len(assets)),
		})
		if err != nil {
			return fmt.Errorf("writing to csv: %w", err)
		}
		w.Flush()
	}

	return nil
}

func assetStr(typ, code, issuer string) string {
	switch typ {
	case "native":
		return "native"
	default:
		return code + ":" + issuer
	}
}

func intsToStrings(i []int32) []string {
	s := make([]string, 0, len(i))
	for _, i := range i {
		s = append(s, strconv.FormatInt(int64(i), 10))
	}
	return s
}
