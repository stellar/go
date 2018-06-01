package resourceadapter

import (
	"context"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/services/horizon/internal/paths"
	. "github.com/stellar/go/protocols/horizon"
)

func PopulatePath(ctx context.Context, dest *Path, q paths.Query, p paths.Path) (err error) {

	dest.DestinationAmount = amount.String(q.DestinationAmount)
	cost, err := p.Cost(q.DestinationAmount)
	if err != nil {
		return
	}

	dest.SourceAmount = amount.String(cost)

	err = p.Source().Extract(
		&dest.SourceAssetType,
		&dest.SourceAssetCode,
		&dest.SourceAssetIssuer)

	if err != nil {
		return
	}

	err = p.Destination().Extract(
		&dest.DestinationAssetType,
		&dest.DestinationAssetCode,
		&dest.DestinationAssetIssuer)

	if err != nil {
		return
	}

	path := p.Path()

	dest.Path = make([]Asset, len(path))

	for i, a := range path {
		err = a.Extract(
			&dest.Path[i].Type,
			&dest.Path[i].Code,
			&dest.Path[i].Issuer)
		if err != nil {
			return
		}
	}

	return
}