package outputs

import "github.com/elastic/elastic-agent-inputs/pkg/publisher"

type noOp struct{}

func NewNoOp() publisher.PipelineV2 {
	return noOp{}
}

func (n noOp) Publish(publisher.Event)      {}
func (n noOp) PublishAll([]publisher.Event) {}
func (n noOp) Close() error                 { return nil }
