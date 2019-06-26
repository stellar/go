package pipeline

import (
	"time"
)

func (p *PipelineNode) Pipe(children ...*PipelineNode) *PipelineNode {
	p.Children = children
	return p
}

func (p *PipelineNode) reset() {
	p.Processor.Reset()

	p.duration = time.Duration(0)
	p.jobs = 0
	p.readEntries = 0
	p.readsPerSecond = 0
	p.queuedEntries = 0
	p.wroteEntries = 0
	p.writesPerSecond = 0
}
