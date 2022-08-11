// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipeline

import (
	"context"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

type pipeline struct {
	logger     *logp.Logger
	output     publisher.PipelineV2
	processors []publisher.Processor
	cancelCtx  context.Context
}

// New instantiates a new publishing pipeline and closes it when the context's done channel is closed
func New(ctx context.Context, logger *logp.Logger, out publisher.Client, processors []publisher.Processor) publisher.PipelineV2 {
	p := pipeline{
		cancelCtx:  ctx,
		logger:     logger,
		output:     out,
		processors: processors,
	}

	go func() {
		<-p.cancelCtx.Done()
		if err := p.Close(); err != nil {
			p.logger.Errorf("could not shutdown pipeline: %s", err)
		}
	}()

	return p
}

// Publish applies all processors to the event, then publishes it
func (p pipeline) Publish(e publisher.Event) {
	var (
		evt = &e
		err = error(nil)
	)

	for _, processor := range p.processors {
		evt, err = processor.Run(evt)
		if err != nil {
			p.logger.Errorf("could not run processor '%s': %s", processor.String(), err.Error())
		}

		if evt == nil {
			p.logger.Debug("dropping event")
			return
		}
	}

	p.output.Publish(*evt)
	p.logger.Debugf("event published")
}

// PublishAll calls Publish for every event
func (p pipeline) PublishAll(events []publisher.Event) {
	for _, e := range events {
		p.Publish(e)
	}
}

func (p pipeline) Close() error {
	if err := p.output.Close(); err != nil {
		p.logger.Errorf("could not shutdown output: %s", err)
		return err
	}

	p.logger.Debug("pipeline shutdown successful")
	return nil
}
