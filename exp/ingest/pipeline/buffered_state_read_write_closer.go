package pipeline

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
)

const bufferSize = 50000

func (b *bufferedStateReadWriteCloser) init() {
	b.buffer = make(chan xdr.LedgerEntry, bufferSize)
}

func (b *bufferedStateReadWriteCloser) GetSequence() uint32 {
	return 0
}

func (b *bufferedStateReadWriteCloser) Read() (xdr.LedgerEntry, error) {
	b.initOnce.Do(b.init)

	entry, more := <-b.buffer
	if more {
		b.readEntries++
		return entry, nil
	} else {
		return xdr.LedgerEntry{}, io.EOF
	}
}

func (b *bufferedStateReadWriteCloser) Write(entry xdr.LedgerEntry) error {
	b.initOnce.Do(b.init)
	b.buffer <- entry
	b.wroteEntries++
	return nil
}

func (b *bufferedStateReadWriteCloser) QueuedEntries() int {
	b.initOnce.Do(b.init)
	return len(b.buffer)
}

func (b *bufferedStateReadWriteCloser) Close() error {
	b.initOnce.Do(b.init)
	close(b.buffer)
	return nil
}

var _ io.StateReader = &bufferedStateReadWriteCloser{}
var _ io.StateWriteCloser = &bufferedStateReadWriteCloser{}
