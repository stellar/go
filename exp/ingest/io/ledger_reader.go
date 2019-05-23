package io

import "github.com/stellar/go/xdr"

type LedgerReader interface {
	GetSequence() uint32
	GetHeader() xdr.LedgerHeader
	Read() (bool, xdr.Transaction, xdr.TransactionResult, xdr.TransactionMeta, error)
}
