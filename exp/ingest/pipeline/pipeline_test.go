package pipeline

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func randomAccountId() xdr.AccountId {
	random, err := keypair.Random()
	if err != nil {
		panic(err)
	}

	id := xdr.AccountId{}
	id.SetAddress(random.Address())
	return id
}

func randomSignerKey() xdr.SignerKey {
	random, err := keypair.Random()
	if err != nil {
		panic(err)
	}

	id := xdr.SignerKey{}
	err = id.SetAddress(random.Address())
	if err != nil {
		panic(err)
	}

	return id
}

func AccountLedgerEntry() xdr.LedgerEntry {
	specialSigner := xdr.SignerKey{}
	err := specialSigner.SetAddress("GCS26OX27PF67V22YYCTBLW3A4PBFAL723QG3X3FQYEL56FXX2C7RX5G")
	if err != nil {
		panic(err)
	}

	signer := specialSigner
	if rand.Int()%100 >= 1 /* % */ {
		signer = randomSignerKey()
	}

	return xdr.LedgerEntry{
		LastModifiedLedgerSeq: 0,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId: randomAccountId(),
				Signers: []xdr.Signer{
					xdr.Signer{
						Key:    signer,
						Weight: 1,
					},
				},
			},
		},
	}
}

func TrustLineLedgerEntry() xdr.LedgerEntry {
	random, err := keypair.Random()
	if err != nil {
		panic(err)
	}

	id := xdr.AccountId{}
	id.SetAddress(random.Address())

	return xdr.LedgerEntry{
		LastModifiedLedgerSeq: 0,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: id,
			},
		},
	}
}

func TestStore(t *testing.T) {
	var s Store

	s.Lock()
	s.Put("value", 0)
	s.Unlock()

	s.Lock()
	v := s.Get("value")
	s.Put("value", v.(int)+1)
	s.Unlock()

	assert.Equal(t, 1, s.Get("value"))
}

func TestBuffer(t *testing.T) {
	buffer := &bufferedStateReadWriteCloser{}
	write := 20
	read := 0

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			_, err := buffer.Read()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					panic(err)
				}
			}
			read++
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < write; i++ {
			buffer.Write(AccountLedgerEntry())
		}
		buffer.Close()
	}()

	wg.Wait()

	assert.Equal(t, 20, read)
}

func ExamplePipeline(t *testing.T) {
	pipeline := &Pipeline{}

	passthroughProcessor := &PassthroughProcessor{}
	accountsOnlyFilter := &EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}
	trustLinesOnlyFilter := &EntryTypeFilter{Type: xdr.LedgerEntryTypeTrustline}
	printCountersProcessor := &PrintCountersProcessor{}
	printAllProcessor := &PrintAllProcessor{}

	pipeline.AddStateProcessorTree(
		pipeline.Node(passthroughProcessor).
			Pipe(
				// Passes accounts only
				pipeline.Node(accountsOnlyFilter).
					Pipe(
						// Finds accounts for a single signer
						pipeline.Node(&AccountsForSignerProcessor{Signer: "GCS26OX27PF67V22YYCTBLW3A4PBFAL723QG3X3FQYEL56FXX2C7RX5G"}).
							Pipe(pipeline.Node(printAllProcessor)),

						// Counts accounts with prefix GA/GB/GC/GD and stores results in a store
						pipeline.Node(&CountPrefixProcessor{Prefix: "GA"}).
							Pipe(pipeline.Node(printCountersProcessor)),
						pipeline.Node(&CountPrefixProcessor{Prefix: "GB"}).
							Pipe(pipeline.Node(printCountersProcessor)),
						pipeline.Node(&CountPrefixProcessor{Prefix: "GC"}).
							Pipe(pipeline.Node(printCountersProcessor)),
						pipeline.Node(&CountPrefixProcessor{Prefix: "GD"}).
							Pipe(pipeline.Node(printCountersProcessor)),
					),
				// Passes trust lines only
				pipeline.Node(trustLinesOnlyFilter).
					Pipe(pipeline.Node(printAllProcessor)),
			),
	)

	buffer := &bufferedStateReadWriteCloser{}

	go func() {
		for i := 0; i < 1000000; i++ {
			buffer.Write(AccountLedgerEntry())
			buffer.Write(TrustLineLedgerEntry())
		}
		buffer.Close()
	}()

	done := pipeline.ProcessState(buffer)
	startTime := time.Now()

	go func() {
		for {
			fmt.Print("\033[H\033[2J")

			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
			fmt.Printf("\tHeapAlloc = %v MiB", bToMb(m.HeapAlloc))
			fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
			fmt.Printf("\tNumGC = %v", m.NumGC)
			fmt.Printf("\tGoroutines = %v", runtime.NumGoroutine())
			fmt.Printf("\tNumCPU = %v\n\n", runtime.NumCPU())

			fmt.Printf("Duration: %s\n\n", time.Since(startTime))

			pipeline.PrintStatus()

			time.Sleep(500 * time.Millisecond)
		}
	}()

	<-done
	time.Sleep(2 * time.Second)
	pipeline.PrintStatus()
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

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

func (n *SimpleProcessor) IncrementAndReturnCallCount() int {
	n.Lock()
	defer n.Unlock()
	n.callCount++
	return n.callCount
}

type PassthroughProcessor struct {
	SimpleProcessor
}

func (p *PassthroughProcessor) ProcessState(store *Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer w.Close()
	defer r.Close()

	for {
		entry, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		err = w.Write(entry)
		if err != nil {
			if err == io.ErrClosedPipe {
				// Reader does not need more data
				return nil
			}
			return err
		}
	}

	return nil
}

func (p *PassthroughProcessor) Name() string {
	return "PassthroughProcessor"
}

func (n *PassthroughProcessor) IsConcurrent() bool {
	return true
}

type EntryTypeFilter struct {
	SimpleProcessor

	Type xdr.LedgerEntryType
}

func (p *EntryTypeFilter) ProcessState(store *Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer w.Close()
	defer r.Close()

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
			err := w.Write(entry)
			if err != nil {
				if err == io.ErrClosedPipe {
					// Reader does not need more data
					return nil
				}
				return err
			}
		}
	}

	return nil
}

func (p *EntryTypeFilter) Name() string {
	return fmt.Sprintf("EntryTypeFilter (%s)", p.Type)
}

type AccountsForSignerProcessor struct {
	SimpleProcessor

	Signer string
}

func (p *AccountsForSignerProcessor) ProcessState(store *Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer w.Close()
	defer r.Close()

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
				err := w.Write(entry)
				if err != nil {
					if err == io.ErrClosedPipe {
						// Reader does not need more data
						return nil
					}
					return err
				}
				break
			}
		}
	}

	return nil
}

func (p *AccountsForSignerProcessor) Name() string {
	return "AccountsForSignerProcessor"
}

type CountPrefixProcessor struct {
	SimpleProcessor
	Prefix string
}

func (p *CountPrefixProcessor) ProcessState(store *Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer w.Close()
	defer r.Close()

	count := 0

	for {
		entry, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		address := entry.Data.Account.AccountId.Address()

		if strings.HasPrefix(address, p.Prefix) {
			err := w.Write(entry)
			if err != nil {
				if err == io.ErrClosedPipe {
					// Reader does not need more data
					return nil
				}
				return err
			}
			count++
		}

		if p.Prefix == "GA" {
			// Make it slower to test full buffer
			// time.Sleep(50 * time.Millisecond)
		}
	}

	store.Lock()
	prevCount := store.Get("count" + p.Prefix)
	if prevCount != nil {
		count += prevCount.(int)
	}
	store.Put("count"+p.Prefix, count)
	store.Unlock()

	return nil
}

func (p *CountPrefixProcessor) IsConcurrent() bool {
	return true
}

func (p *CountPrefixProcessor) Name() string {
	return fmt.Sprintf("CountPrefixProcessor (%s)", p.Prefix)
}

type PrintCountersProcessor struct {
	SimpleProcessor
}

func (p *PrintCountersProcessor) ProcessState(store *Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer w.Close()
	defer r.Close()

	// TODO, we should use context with cancel and value to check when pipeline is done.
	for {
		_, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
	}

	if p.IncrementAndReturnCallCount() != 4 {
		return nil
	}

	store.Lock()
	fmt.Println("countGA", store.Get("countGA"))
	fmt.Println("countGB", store.Get("countGB"))
	fmt.Println("countGC", store.Get("countGC"))
	fmt.Println("countGD", store.Get("countGD"))
	store.Unlock()

	return nil
}

func (p *PrintCountersProcessor) Name() string {
	return "PrintCountersProcessor"
}

type PrintAllProcessor struct {
	SimpleProcessor
}

func (p *PrintAllProcessor) ProcessState(store *Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer w.Close()
	defer r.Close()

	entries := 0
	for {
		_, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		entries++
		// fmt.Printf("%+v\n", entry)
	}

	fmt.Printf("Found %d entries\n", entries)

	return nil
}

func (p *PrintAllProcessor) Name() string {
	return "PrintAllProcessor"
}
