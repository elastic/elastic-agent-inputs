// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

type loadGenerator struct {
	cfg    Config
	now    func() time.Time
	logger *logp.Logger

	output publisher.Client
}

func (l loadGenerator) Run(ctx context.Context) error {
	t := l.now()
	delta := l.cfg.TimeDelta
	delay := l.cfg.Delay

	// Loop forever
	if l.cfg.Loop {
		for {
			select {
			case <-ctx.Done():
				l.logger.Info("Shutdown signal received, closing loadGenerator")
				return nil
			default:
				if err := l.send(l.next(t)); err != nil {
					return fmt.Errorf("error sending event: %w", err)
				}
				if l.cfg.CurrentTime {
					t = l.now()
				} else {
					t = t.Add(delta)
				}
				time.Sleep(delay)
			}
		}
	}

	// Generate a set number of events
	for i := uint64(0); i < l.cfg.EventsCount; i++ {
		if err := l.send(l.next(t)); err != nil {
			return fmt.Errorf("error sending event: %w", err)
		}
		if l.cfg.CurrentTime {
			t = l.now()
		} else {
			t = t.Add(delta)
		}
		time.Sleep(delay)
	}

	return nil
}

// next returns the next event
func (l loadGenerator) next(t time.Time) string {
	return NewJSONLogFormat(t)
}

// send sends the event to the publishing pipeline
// TODO (Tiago): implement it
func (l loadGenerator) send(event string) error {
	e := publisher.Event{
		ShipperMessage: &messages.Event{
			Fields: &messages.Struct{
				Data: map[string]*messages.Value{
					"message": helpers.NewStringValue(event),
				},
			},
		},
	}

	l.output.Publish(e)
	return nil
}
