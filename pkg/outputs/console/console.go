// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package console

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher/acker"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
)

// Pipeline implements pkg/publisher.Pipeline
type Pipeline struct {
	cancelCtx ctxtool.CancelContext
	logger    *logp.Logger
	out       io.Writer
	client    publisher.Client
}

// Client implements pkg/publisher.Client
type Client struct {
	out       io.Writer
	cancelCtx ctxtool.CancelContext
	logger    *logp.Logger
	Acker     publisher.ACKer
}

func NewPipeline(ctx context.Context, logger *logp.Logger, out io.Writer) Pipeline {
	cancelCtx := ctxtool.WithCancelContext(ctx)
	p := Pipeline{
		logger:    logger,
		out:       out,
		cancelCtx: cancelCtx,
	}

	go func() {
		<-ctx.Done()
		logger.Debug("got done signal, shutting down pipeline")
		if err := p.Cancel(); err != nil {
			p.logger.Errorf("cold not shutdown pipeline: %s", err)
		}
	}()

	return p
}

func (p Pipeline) Cancel() error {
	defer p.cancelCtx.Cancel()
	if p.client != nil {
		if err := p.client.Close(); err != nil {
			p.logger.Debugf("error closing client: %s", err)
			return fmt.Errorf("could not close client: %w", err)
		}
	}

	return nil
}

func (p Pipeline) newClient(cfg publisher.ClientConfig, logger *logp.Logger) (publisher.Client, error) {
	c := Client{
		logger:    logger,
		Acker:     cfg.ACKHandler,
		out:       p.out,
		cancelCtx: ctxtool.WithCancelContext(p.cancelCtx.Context),
	}

	go func() {
		<-c.cancelCtx.Done()
		if err := c.Close(); err != nil {
			c.logger.Errorf("error shutingdown client: %s", err)
		}
	}()

	if c.Acker == nil {
		c.Acker = acker.NoOp()
	}

	return c, nil
}

// ConnectWith returns a Client using the given configuration
func (p Pipeline) ConnectWith(cfg publisher.ClientConfig) (publisher.Client, error) {
	return p.newClient(cfg, p.logger.Named("client"))

}

// Connect returns a client using the default configuration
func (p Pipeline) Connect() (publisher.Client, error) {
	return p.ConnectWith(publisher.ClientConfig{})
}

// Publish publishes a single event to stdout
func (c Client) Publish(event publisher.Event) {
	if c.Acker != nil {
		c.Acker.AddEvent(event, true)
	}

	if err := c.publish(event); err != nil {
		c.logger.Errorf("could not publish event: %s", err)
		c.logger.Debugf("event not published: '%s'", event.ShipperMessage.String())
	}

	if c.Acker != nil {
		c.Acker.ACKEvents(1)
	}
}

// PublishAll calls c.publish for every event then
// ack them all at once
func (c Client) PublishAll(events []publisher.Event) {
	for _, event := range events {
		c.publish(event)
	}

	if c.Acker != nil {
		c.Acker.ACKEvents(len(events))
	}
}

// Close no op function, stdout does not need closing
func (c Client) Close() error {
	return nil
}

// publish publishes an event
func (c Client) publish(event publisher.Event) error {
	data, err := protojson.Marshal(event.ShipperMessage)
	if err != nil {
		return fmt.Errorf("could not marshal event as JSON: %w", err)
	}

	data = append(data, []byte("\n")...)

	if _, err := c.out.Write(data); err != nil {
		return fmt.Errorf("could not write to output: %w", err)
	}

	return nil
}
