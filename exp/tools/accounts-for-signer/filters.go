package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/xdr"
)

type SimpleProcessor struct {
	sync.Mutex
	callCount int
}

func (n *SimpleProcessor) IsConcurrent() bool {
	return false
}

func (n *SimpleProcessor) RequiresInput() bool {
	return true
}

func (n *SimpleProcessor) CallCount() int {
	n.Lock()
	defer n.Unlock()
	n.callCount++
	return n.callCount
}

type EntryTypeFilter struct {
	SimpleProcessor

	Type xdr.LedgerEntryType
}

func (p *EntryTypeFilter) ProcessState(store *pipeline.Store, r io.StateReader, w io.StateWriteCloser) error {
	for {
		entry, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		if entry.Data.Type == p.Type {
			w.Write(entry)
		}
	}

	w.Close()
	return nil
}

func (p *EntryTypeFilter) Name() string {
	return fmt.Sprintf("EntryTypeFilter (%s)", p.Type)
}

type AccountsForSignerProcessor struct {
	SimpleProcessor

	Signer string
}

func (p *AccountsForSignerProcessor) ProcessState(store *pipeline.Store, r io.StateReader, w io.StateWriteCloser) error {
	for {
		entry, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		if entry.Data.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		for _, signer := range entry.Data.Account.Signers {
			if signer.Key.Address() == p.Signer {
				w.Write(entry)
				break
			}
		}
	}

	w.Close()
	return nil
}

func (p *AccountsForSignerProcessor) Name() string {
	return "AccountsForSignerProcessor"
}

func (n *AccountsForSignerProcessor) IsConcurrent() bool {
	return true
}

type PrintAllProcessor struct {
	SimpleProcessor
	Filename string
}

func (p *PrintAllProcessor) ProcessState(store *pipeline.Store, r io.StateReader, w io.StateWriteCloser) error {
	defer w.Close()

	f, err := os.Create(p.Filename)
	if err != nil {
		return err
	}

	defer f.Close()

	entries := 0
	for {
		entry, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		entries++
		switch entry.Data.Type {
		case xdr.LedgerEntryTypeAccount:
			fmt.Fprintf(
				f,
				"%s,%d,%d\n",
				entry.Data.Account.AccountId.Address(),
				entry.Data.Account.Balance,
				entry.Data.Account.SeqNum,
			)
		default:
			// Ignore for now
		}
	}

	return nil
}

func (p *PrintAllProcessor) Name() string {
	return "PrintAllProcessor"
}
