package pipeline

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stellar/go/exp/ingest/io"
)

func (p *Pipeline) Node(processor StateProcessor) *PipelineNode {
	return &PipelineNode{
		Processor: processor,
	}
}

func (p *Pipeline) PrintStatus() {
	p.printNodeStatus(p.rootStateProcessor, 0)
}

func (p *Pipeline) printNodeStatus(node *PipelineNode, level int) {
	fmt.Print(strings.Repeat("  ", level))

	var wrRatio = float32(0)
	if node.readEntries > 0 {
		wrRatio = float32(node.wroteEntries) / float32(node.readEntries)
	}

	icon := ""
	if node.queuedEntries > bufferSize/10*9 {
		icon = "⚠️ "
	}

	fmt.Printf(
		"└ %s%s read=%d (queued=%d rps=%d) wrote=%d (w/r ratio = %1.5f) concurrent=%t jobs=%d\n",
		icon,
		node.Processor.Name(),
		node.readEntries,
		node.queuedEntries,
		node.readsPerSecond,
		node.wroteEntries,
		wrRatio,
		node.Processor.IsConcurrent(),
		node.jobs,
	)

	if node.jobs > 1 {
		fmt.Print(strings.Repeat("  ", level))
		fmt.Print("  ")
		for i := 0; i < node.jobs; i++ {
			fmt.Print("• ")
		}
		fmt.Println("")
	}

	for _, child := range node.Children {
		p.printNodeStatus(child, level+1)
	}
}

func (p *Pipeline) AddStateProcessorTree(rootProcessor *PipelineNode) {
	p.rootStateProcessor = rootProcessor
}

func (p *Pipeline) ProcessState(readCloser io.StateReadCloser) (done chan error) {
	return p.processStateNode(&Store{}, p.rootStateProcessor, readCloser)
}

func (p *Pipeline) processStateNode(store *Store, node *PipelineNode, readCloser io.StateReadCloser) chan error {
	outputs := make([]io.StateWriteCloser, len(node.Children))

	for i := range outputs {
		outputs[i] = &bufferedStateReadWriteCloser{}
	}

	var wg sync.WaitGroup

	jobs := 1
	if node.Processor.IsConcurrent() {
		jobs = 20
	}

	node.jobs = jobs

	writeCloser := &multiWriteCloser{
		writers:    outputs,
		closeAfter: jobs,
	}

	for i := 1; i <= jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := node.Processor.ProcessState(store, readCloser, writeCloser)
			if err != nil {
				// TODO return to pipeline error channel
				panic(err)
			}
		}()
	}

	go func() {
		// Update stats
		for {
			// This is not thread-safe: check if Mutex slows it down a lot...
			readBuffer, readBufferIsBufferedStateReadWriteCloser := readCloser.(*bufferedStateReadWriteCloser)
			writeBuffer := writeCloser

			interval := time.Second

			node.writesPerSecond = (writeBuffer.wroteEntries - node.wroteEntries) * int(time.Second/interval)
			node.wroteEntries = writeBuffer.wroteEntries

			if readBufferIsBufferedStateReadWriteCloser {
				node.readsPerSecond = (readBuffer.readEntries - node.readEntries) * int(time.Second/interval)
				node.readEntries = readBuffer.readEntries
				node.queuedEntries = readBuffer.QueuedEntries()
			}

			time.Sleep(interval)
		}
	}()

	for i, child := range node.Children {
		wg.Add(1)
		go func(i int, child *PipelineNode) {
			defer wg.Done()
			done := p.processStateNode(store, child, outputs[i].(*bufferedStateReadWriteCloser))
			<-done
		}(i, child)
	}

	done := make(chan error)

	go func() {
		wg.Wait()
		done <- nil
	}()

	return done
}
