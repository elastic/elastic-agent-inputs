// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.uber.org/atomic"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

type Input struct {
	cfg      Config
	now      func() time.Time
	logger   *logp.Logger
	output   publisher.PipelineV2
	unit     *client.Unit
	streamID string

	events atomic.Uint64
}

func NewLoadGenerator(
	logger *logp.Logger,
	output publisher.PipelineV2,
	source *structpb.Struct,
	unit *client.Unit,
	streamID string) (input.InputV2, error) {

	cfg, err := configFromSource(source)
	if err != nil {
		return nil, fmt.Errorf("could not parse config from source: %w", err)
	}

	return &Input{
		now:      time.Now,
		logger:   logger,
		cfg:      cfg,
		unit:     unit,
		streamID: streamID,

		output: output,
	}, nil
}

func (l *Input) Stop(ctx context.Context) error {
	// nothing here, for now
	return nil
}

// Run a single instance of the load generator
func (l *Input) Run(stopChan chan struct{}) error {
	timeDiff := l.now()
	delta := l.cfg.TimeDelta
	delay := l.cfg.Delay
	gotStop := false
	evts := uint64(0)
	defer func() {
		l.events.Add(evts)
		if !gotStop {
			_ = l.unit.UpdateState(client.UnitStateHealthy, fmt.Sprintf("input for stream %s is ending early", l.streamID), nil)
			l.logger.Debugf("runner %s returned early", l.streamID)
		}
	}()

	l.logger.Infof("Starting load gen loop for stream %s: loop forever: %v; iterations: %d", l.streamID, l.cfg.Loop, l.cfg.EventsCount)
	for {
		// if l.cfg.Loop is set, loop forever. Otherwise, write l.cfg.EventsCount events
		if !l.cfg.Loop && evts == l.cfg.EventsCount {
			l.logger.Debugf("Sent %d events, stopping loadGenerator for stream ID %s", evts, l.streamID)
			break
		}
		select {
		case <-stopChan:
			l.logger.Debugf("Shutdown signal received, closing loadGenerator for stream %s, wrote %d events", l.streamID, evts)
			gotStop = true
			return nil
		default:
			if err := l.send(l.next(timeDiff)); err != nil {
				return fmt.Errorf("error sending event: %w", err)
			}
			evts++
			if l.cfg.CurrentTime {
				timeDiff = l.now()
			} else {
				timeDiff = timeDiff.Add(delta)
			}
			time.Sleep(delay)
		}
	}
	return nil
}

// next returns the next event
func (l *Input) next(t time.Time) string {
	return NewJSONLogFormat(t)
}

// send sends the event to the publishing pipeline
func (l *Input) send(event string) error {
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
