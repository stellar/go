package simplepath

import (
	"errors"
	"math/big"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/xdr"
)

// ErrNotEnough represents an error that occurs when pricing a trade on an
// orderbook.  This error occurs when the orderbook cannot fulfill the
// requested amount.
var ErrNotEnough = errors.New("not enough depth")

// orderbook represents a one-way orderbook that is selling you a specific asset (ob.Selling)
type orderBook struct {
	Selling xdr.Asset // the offers are selling this asset
	Buying  xdr.Asset // the offers are buying this asset
	Q       *core.Q
}

// CostToConsumeLiquidity returns the buyingAmount (ob.Buying) needed to consume the sellingAmount (ob.Selling)
func (ob *orderBook) CostToConsumeLiquidity(sellingAmount xdr.Int64) (xdr.Int64, error) {
	// load orderbook from core's db
	sql, e := ob.query()
	if e != nil {
		return 0, e
	}
	rows, e := ob.Q.Query(sql)
	if e != nil {
		return 0, e
	}
	defer rows.Close()

	// remaining is the units of ob.Selling that we want to consume
	remaining := int64(sellingAmount)
	var buyingAmount int64
	for rows.Next() {
		// load data from the row
		var offerAmount, pricen, priced, offerid int64
		e = rows.Scan(&offerAmount, &pricen, &priced, &offerid)
		if e != nil {
			return 0, e
		}

		if offerAmount >= remaining {
			buyingAmount += mul(remaining, pricen, priced)
			return xdr.Int64(buyingAmount), nil
		}

		buyingAmount += mul(offerAmount, pricen, priced)
		remaining -= offerAmount
	}
	return 0, ErrNotEnough
}

func (ob *orderBook) query() (sq.SelectBuilder, error) {
	var (
		// selling/buying types
		st, bt xdr.AssetType
		// selling/buying codes
		sc, bc string
		// selling/buying issuers
		si, bi string
	)
	e := ob.Selling.Extract(&st, &sc, &si)
	if e != nil {
		return sq.SelectBuilder{}, e
	}
	e = ob.Buying.Extract(&bt, &bc, &bi)
	if e != nil {
		return sq.SelectBuilder{}, e
	}

	sql := sq.
		Select("amount", "pricen", "priced", "offerid").
		From("offers").
		Where(sq.Eq{
			"sellingassettype":               st,
			"COALESCE(sellingassetcode, '')": sc,
			"COALESCE(sellingissuer, '')":    si}).
		Where(sq.Eq{
			"buyingassettype":               bt,
			"COALESCE(buyingassetcode, '')": bc,
			"COALESCE(buyingissuer, '')":    bi}).
		OrderBy("price ASC")
	return sql, nil
}

// mul multiplies the input amount by the input price
func mul(amount int64, pricen int64, priced int64) int64 {
	var r, n, d big.Int

	r.SetInt64(amount)
	n.SetInt64(pricen)
	d.SetInt64(priced)

	r.Mul(&r, &n)
	r.Quo(&r, &d)
	return r.Int64()
}
