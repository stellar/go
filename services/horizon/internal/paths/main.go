package paths

import (
	"github.com/stellar/go/xdr"
)

// Query is a query for paths
type Query struct {
	DestinationAddress string
	DestinationAsset   xdr.Asset
	DestinationAmount  xdr.Int64
	SourceAssets       []xdr.Asset
}

// Path is the result returned by a path finder and is tied to the DestinationAmount used in the input query
type Path struct {
	Path        []xdr.Asset
	Source      xdr.Asset
	Destination xdr.Asset
	Cost        xdr.Int64 // delimited in the Source assets, to be used as `sendMax` field for a `PathPaymentOp` struct.
}

// Finder finds paths.
type Finder interface {
	Find(Query) ([]Path, error)
}
