package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stellar/go/support/errors"
)

func New(rootProcessor *PipelineNode) *Pipeline {
	return &Pipeline{root: rootProcessor}
}

func Node(processor Processor) *PipelineNode {
	return &PipelineNode{
		Processor: processor,
	}
}

func (p *Pipeline) PrintStatus() {
	p.printNodeStatus(p.root, 0)
}

func (p *Pipeline) AddPreProcessingHook(hook func(context.Context) error) {
	p.preProcessingHooks = append(p.preProcessingHooks, hook)
}

func (p *Pipeline) AddPostProcessingHook(hook func(context.Context, error) error) {
	p.postProcessingHooks = append(p.postProcessingHooks, hook)
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

func (p *Pipeline) SetRoot(rootProcessor *PipelineNode) {
	p.root = rootProcessor
}

// setRunning protects from processing more than once at a time.
func (p *Pipeline) setRunning(setRunning bool) {
	if setRunning {
		if p.running {
			panic("Pipeline is running...")
		}

		if p.shutDown {
			panic("Pipeline was shut down...")
		}
	}

	p.running = setRunning
}

// reset resets internal state of the pipeline and all the nodes and processors.
func (p *Pipeline) reset() {
	p.cancelled = false
	p.resetNode(p.root)
}

func (p *Pipeline) sendPreProcessingHooks(ctx context.Context) error {
	for _, hook := range p.preProcessingHooks {
		err := hook(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pipeline) sendPostProcessingHooks(ctx context.Context, processingError error) error {
	for _, hook := range p.postProcessingHooks {
		err := hook(ctx, processingError)
		if err != nil {
			return err
		}
	}

	return nil
}

// resetNode resets internal state of the pipeline node and internal processor and
// calls itself recursively on all of the children.
func (p *Pipeline) resetNode(node *PipelineNode) {
	node.reset()
	for _, child := range node.Children {
		p.resetNode(child)
	}
}

func (p *Pipeline) Process(readCloser ReadCloser) <-chan error {
	// Protects internal fields
	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	p.setRunning(true)
	p.reset()

	err := p.sendPreProcessingHooks(readCloser.GetContext())
	if err != nil {
		p.setRunning(false)
		errorChan := make(chan error)
		errorChan <- errors.Wrap(err, "Error running pre-hook")
		return errorChan
	}

	var ctx context.Context
	ctx, p.cancelFunc = context.WithCancel(context.Background())
	return p.processStateNode(ctx, &Store{}, p.root, readCloser)
}

func (p *Pipeline) processStateNode(ctx context.Context, store *Store, node *PipelineNode, readCloser ReadCloser) <-chan error {
	outputs := make([]WriteCloser, len(node.Children))

	for i := range outputs {
		outputs[i] = &BufferedReadWriteCloser{
			context: readCloser.GetContext(),
		}
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

	var processingError error

	for i := 1; i <= jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := node.Processor.Process(ctx, store, readCloser, writeCloser)
			if err != nil {
				// Protects from cancelling twice and sending multiple errors to err channel
				p.cancelledMutex.Lock()
				defer p.cancelledMutex.Unlock()

				if p.cancelled {
					return
				}

				wrappedErr := errors.Wrap(err, fmt.Sprintf("Processor %s errored", node.Processor.Name()))

				p.cancelled = true
				p.cancelFunc()
				processingError = wrappedErr
			}
		}()
	}

	finishUpdatingStats := p.updateStats(node, readCloser, writeCloser)

	for i, child := range node.Children {
		wg.Add(1)
		go func(i int, child *PipelineNode) {
			defer wg.Done()
			err := <-p.processStateNode(ctx, store, child, outputs[i].(*BufferedReadWriteCloser))
			if err != nil {
				processingError = err
			}
		}(i, child)
	}

	errorChan := make(chan error)

	go func() {
		wg.Wait()
		finishUpdatingStats <- true

		if node == p.root {
			// If pipeline processing is finished run post-hooks (if not shut down)
			// and send error if not already sent.
			var returnError error
			if p.shutDown {
				returnError = nil
			} else {
				err := p.sendPostProcessingHooks(readCloser.GetContext(), processingError)
				if err != nil {
					returnError = errors.Wrap(err, "Error running post-hook")
				} else {
					returnError = processingError
				}
			}

			p.runningMutex.Lock()
			p.setRunning(false)
			p.runningMutex.Unlock()

			errorChan <- returnError
		} else {
			// For non-root node just send an error
			errorChan <- processingError
		}
	}()

	return errorChan
}

// Shutdown stops the processing. Please note that post-processing hooks will not
// be executed when Shutdown() is called.
func (p *Pipeline) Shutdown() {
	// Protects from cancelling twice
	p.cancelledMutex.Lock()
	defer p.cancelledMutex.Unlock()

	// Protects internal fields
	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if p.cancelled {
		return
	}
	p.shutDown = true
	p.cancelled = true
	p.cancelFunc()
}

func (p *Pipeline) updateStats(node *PipelineNode, readCloser ReadCloser, writeCloser *multiWriteCloser) chan<- bool {
	// Update stats
	interval := time.Second
	done := make(chan bool)
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			// This is not thread-safe: check if Mutex slows it down a lot...
			readBuffer, readBufferIsBufferedReadWriteCloser := readCloser.(*BufferedReadWriteCloser)

			node.writesPerSecond = (writeCloser.wroteEntries - node.wroteEntries) * int(time.Second/interval)
			node.wroteEntries = writeCloser.wroteEntries

			if readBufferIsBufferedReadWriteCloser {
				node.readsPerSecond = (readBuffer.readEntries - node.readEntries) * int(time.Second/interval)
				node.readEntries = readBuffer.readEntries
				node.queuedEntries = readBuffer.QueuedEntries()
			}

			select {
			case <-ticker.C:
				continue
			case <-done:
				// Pipeline done
				return
			}
		}
	}()

	return done
}
