package pipeline

import (
	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
)

func LedgerNode(processor LedgerProcessor) *supportPipeline.PipelineNode {
	return &supportPipeline.PipelineNode{
		Processor: &ledgerProcessorWrapper{processor},
	}
}

func (p *LedgerPipeline) Process(reader io.LedgerReader) <-chan error {
	return p.Pipeline.Process(&ledgerReaderWrapper{reader})
}
