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
			// We can keep sending entries even when io.ErrClosedPipe is returned
			// as bufferedStateReadWriteCloser will ignore them (won't add them to
			// a channel).
			results <- w.Write(entry)
		}(w)
	}

	wg.Wait()

	countClosedPipes := 0
	for range m.writers {
		err := <-results
		if err != nil {
			if err == io.ErrClosedPipe {
				countClosedPipes++
			} else {
				return err
			}
		}
	}

	// When all pipes are closed return `io.ErrClosedPipe` because there are no
	// active readers anymore.
	if countClosedPipes == len(m.writers) {
		return io.ErrClosedPipe
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
