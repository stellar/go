package pipeline

import (
	"sync"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
)

func (m *multiWriteCloser) Write(entry xdr.LedgerEntry) error {
	m.mutex.Lock()
	m.wroteEntries++
	m.mutex.Unlock()

	var wg sync.WaitGroup
	results := make(chan error, len(m.writers))

	for _, w := range m.writers {
		wg.Add(1)
		go func(w io.StateWriteCloser) {
			defer wg.Done()
			err := w.Write(entry)
			if err != nil {
				results <- err
			} else {
				results <- nil
			}
		}(w)
	}

	wg.Wait()

	for range m.writers {
		err := <-results
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *multiWriteCloser) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.closeAfter--
	if m.closeAfter > 0 {
		return nil
	}

	for _, w := range m.writers {
		err := w.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
