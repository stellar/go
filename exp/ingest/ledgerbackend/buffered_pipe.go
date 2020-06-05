package ledgerbackend

type bufferedPipe struct {
	buffer chan byte
}

func newBufferedPipe(cap int) *bufferedPipe {
	return &bufferedPipe{
		buffer: make(chan byte, cap),
	}
}

func (b *bufferedPipe) Read(p []byte) (n int, err error) {
	if len(b.buffer) == 0 {
		return 0, nil
	}

	read := 0
	for i := range p {
		if len(b.buffer) == 0 {
			break
		}

		p[i] = <-b.buffer
		read++
	}

	return read, nil
}

func (b *bufferedPipe) Write(p []byte) (n int, err error) {
	for _, by := range p {
		b.buffer <- by
	}

	return len(p), nil
}

func (b *bufferedPipe) IsEmpty() bool {
	return len(b.buffer) == 0
}
