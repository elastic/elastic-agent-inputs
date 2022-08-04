// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

type loadGenerator struct {
	cfg    Config
	now    func() time.Time
	logger *logp.Logger
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
	_, err := os.Stdout.Write([]byte(event + "\n"))
	return err
}
