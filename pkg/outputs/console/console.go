// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package console

import (
	"context"
	"fmt"
	"io"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher/acker"
	"github.com/elastic/elastic-agent-libs/logp"
	"google.golang.org/protobuf/encoding/protojson"
)

type Config struct {
	Enabled bool `yaml:"enabled"`
	Pretty  bool `yaml:"pretty"`
}

// client implements pkg/publisher.client
type client struct {
	out       io.Writer
	cancelCtx context.Context
	logger    *logp.Logger
	Acker     publisher.ACKer
	marshaler protojson.MarshalOptions
}

// New returns a new console client that implements publisher.PipelineV2
func New(ctx context.Context, out io.Writer, logger *logp.Logger, ackerInstance publisher.ACKer, cfg Config) publisher.PipelineV2 {
	marshaller := protojson.MarshalOptions{
		AllowPartial:  true,
		UseProtoNames: true,
		Multiline:     cfg.Pretty,
	}

	if !cfg.Enabled {
		out = io.Discard
	}

	if ackerInstance == nil {
		ackerInstance = acker.NoOp()
	}

	return client{
		out:       out,
		logger:    logger,
		Acker:     ackerInstance,
		cancelCtx: ctx,
		marshaler: marshaller,
	}
}

// Publish publishes a single event to stdout
func (c client) Publish(event publisher.Event) {
	c.Acker.AddEvent(event, true)

	if err := c.publish(event); err != nil {
		c.logger.Errorf("could not publish event: %s", err)
		c.logger.Debugf("event not published: '%s'", event.ShipperMessage.String())
	}

	c.Acker.ACKEvents(1)
	c.logger.Debugf("event published")
}

// PublishAll calls c.publish for every event then
// ack them all at once
func (c client) PublishAll(events []publisher.Event) {
	for _, event := range events {
		c.publish(event)
	}

	c.Acker.ACKEvents(len(events))
}

// Close no op function, stdout does not need closing
func (c client) Close() error {
	c.logger.Debug("shutdown done")
	return nil
}

// publish publishes an event
func (c client) publish(event publisher.Event) error {
	data, err := c.marshaler.Marshal(event.ShipperMessage)
	if err != nil {
		return fmt.Errorf("could not marshal event as JSON: %w", err)
	}

	data = append(data, []byte("\n")...)

	if _, err := c.out.Write(data); err != nil {
		return fmt.Errorf("could not write to output: %w", err)
	}

	return nil
}
