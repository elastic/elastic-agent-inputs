// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.uber.org/atomic"
	"google.golang.org/protobuf/types/known/structpb"
)

type loadGenerator struct {
	now     func() time.Time
	logger  *logp.Logger
	runners map[string]*loadRunner
	runMut  *sync.Mutex
	events  atomic.Uint64
}

func newLoadGenerator(logger *logp.Logger) *loadGenerator {

	return &loadGenerator{
		now:     time.Now,
		logger:  logger,
		runners: make(map[string]*loadRunner),
		runMut:  &sync.Mutex{},
	}
}

// load runner handles the communications between a loadGenerator instance and the server
type loadRunner struct {
	stopFuncs map[string]chan struct{}
	mapMut    *sync.Mutex
	unit      *client.Unit
}

func newLoadRunner() *loadRunner {
	return &loadRunner{
		stopFuncs: make(map[string]chan struct{}),
		mapMut:    &sync.Mutex{},
	}
}

func (l *loadGenerator) Close(ctx context.Context) error {
	// nothing here, for now
	return nil
}

// Start the generator and initialize the client
func (l *loadGenerator) Start(ctx context.Context) error {
	agentClient, _, err := client.NewV2FromReader(os.Stdin, client.VersionInfo{
		Name:    "beat-v2-client",
		Version: "alpha",
		Meta:    map[string]string{},
	})
	if err != nil {
		return fmt.Errorf("error fetching client from stdin: %w", err)
	}
	return l.StartWithClient(ctx, agentClient)
}

// StartWithClient initializes the client and starts the main runloop of the generator
func (l *loadGenerator) StartWithClient(ctx context.Context, agentClient client.V2) error {
	err := agentClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting client: %w", err)
	}
	l.logger.Debugf("Started agent client, listening for events")

	for {
		select {
		case <-ctx.Done():
			l.logger.Debugf("Got context done in StartWithClient")
			return nil
		case change := <-agentClient.UnitChanges():
			switch change.Type {
			case client.UnitChangedAdded:
				l.logger.Debugf("Got new Unit")
				err := l.handleNewUnit(change.Unit)
				if err != nil {
					_ = change.Unit.UpdateState(client.UnitStateFailed, fmt.Sprintf("New unit failed with: %s", err), nil)
				}
			case client.UnitChangedModified:
				l.logger.Debugf("Got unit modified")
				l.handleUnitModified(change.Unit)
			case client.UnitChangedRemoved:
				l.logger.Debugf("Got unit removed for ID %s", change.Unit.ID())
				l.removeUnit(change.Unit)
				if l.checkDone() {
					l.logger.Debugf("All active units removed, exiting")
					l.logger.Infof("Wrote a total of %d Events", l.events.Load())
					return nil
				}
			}

		}
	}

}

// checkDone checks to see if we have any running units.
func (l *loadGenerator) checkDone() bool {
	l.runMut.Lock()
	defer l.runMut.Unlock()
	return len(l.runners) == 0
}

// removeUnit deletes a unit from the map. This should be called after we've removed any streams associated with a unit.
func (l *loadGenerator) removeUnit(unit *client.Unit) {
	l.runMut.Lock()
	defer l.runMut.Unlock()
	delete(l.runners, unit.ID())
}

// handleUnitModified wraps any functions needed to manage changes to a unit
func (l *loadGenerator) handleUnitModified(unit *client.Unit) {
	state, _, _ := unit.Expected()
	if state == client.UnitStateStopping || state == client.UnitStateStopped {
		l.tearDownUnit(unit)
	} else {
		// Not sure how this should respond to unit changes.
		// Reload the entire unit? Diff all the streams for changes, and reload individual streams?
		l.logger.Debugf("Got unit state: %s", state)
	}
}

// tearDownUnit stops a unit, and all streams associated with it
func (l *loadGenerator) tearDownUnit(unit *client.Unit) {
	l.runMut.Lock()
	defer l.runMut.Unlock()

	_ = unit.UpdateState(client.UnitStateStopping, "starting unit shutdown", nil)

	id := unit.ID()
	unitRunner := l.runners[id]
	unitRunner.mapMut.Lock()
	for streamID, runner := range unitRunner.stopFuncs {
		l.logger.Debugf("Stopping runner %s", streamID)
		_ = unit.UpdateState(client.UnitStateStopping, fmt.Sprintf("sending shutdown to unit %s", streamID), nil)
		runner <- struct{}{}
		// remove the stream from our map
		delete(unitRunner.stopFuncs, streamID)
	}
	unitRunner.mapMut.Unlock()

	l.logger.Debugf("All runners for unit %s have stopped", id)
	_ = unit.UpdateState(client.UnitStateStopped, "all runners for unit have stopped", nil)
}

// handleNewUnit starts a new unit and initializes a collection of runners from the config
func (l *loadGenerator) handleNewUnit(unit *client.Unit) error {
	l.runMut.Lock()
	defer l.runMut.Unlock()
	// As far as configuration: each stream will be treated as a different load-generator run,
	// This way the config can easily scale up the number of parallel runs.
	_, _, cfg := unit.Expected()
	mgr := newLoadRunner()
	_ = unit.UpdateState(client.UnitStateConfiguring, "configuring unit", nil)
	for _, runner := range cfg.Streams {
		streamID := runner.Id
		cfg, err := setConfigValues(runner.Source)
		if err != nil {
			return fmt.Errorf("error configuring runner: %w", err)
		}
		stop := make(chan struct{}, 1)
		_ = unit.UpdateState(client.UnitStateStarting, fmt.Sprintf("Starting stream with ID %s", streamID), nil)
		go func() {
			err = l.Run(stop, cfg, unit, streamID)
			if err != nil {
				_ = unit.UpdateState(client.UnitStateDegraded, fmt.Sprintf("Error running stream with ID %s: %s", streamID, err), nil)
			}
		}()

		mgr.stopFuncs[streamID] = stop
	}

	mgr.unit = unit
	l.runners[unit.ID()] = mgr
	_ = unit.UpdateState(client.UnitStateHealthy, "started runners", nil)
	return nil
}

// setConfigValues creates the config struct from the protobuf source map
func setConfigValues(source *structpb.Struct) (Config, error) {
	result := DefaultConfig()
	cfg, err := config.NewConfigFrom(source.AsMap())
	if err != nil {
		return result, fmt.Errorf("error creating new config: %w", err)
	}
	err = cfg.Unpack(&result)
	if err != nil {
		return result, fmt.Errorf("error unpacking config: %w", err)
	}
	return result, nil
}

// Run a single instance of the load generator
func (l *loadGenerator) Run(stopChan chan struct{}, cfg Config, unit *client.Unit, streamID string) error {
	timeDiff := l.now()
	delta := cfg.TimeDelta
	delay := cfg.Delay
	gotStop := false
	evts := uint64(0)
	defer func() {
		l.events.Add(evts)
		if !gotStop {
			_ = unit.UpdateState(client.UnitStateHealthy, fmt.Sprintf("runner for stream %s is ending early", streamID), nil)
			l.logger.Debugf("runner %s returned early", streamID)
		}
	}()

	// setup the output
	outFile := os.Stdout
	if cfg.OutFile != "" {
		f, err := os.OpenFile(cfg.OutFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("error creating file at %s: %w", cfg.OutFile, err)
		}
		outFile = f
	}
	defer outFile.Close()

	l.logger.Debugf("Starting load gen loop for stream %s: Output: %s; loop forever: %v; iterations: %d", streamID, cfg.OutFile, cfg.Loop, cfg.EventsCount)
	for {
		// if cfg.Loop is set, loop forever. Otherwise, write cfg.EventsCount events
		if !cfg.Loop && evts == cfg.EventsCount {
			l.logger.Debugf("Sent %d events, stopping loadGenerator for stream ID %s", evts, streamID)
			break
		}
		select {
		case <-stopChan:
			l.logger.Debugf("Shutdown signal received, closing loadGenerator for stream %s, wrote %d events", streamID, evts)
			gotStop = true
			return nil
		default:
			if err := l.send(l.next(timeDiff), outFile); err != nil {
				return fmt.Errorf("error sending event: %w", err)
			}
			evts++
			if cfg.CurrentTime {
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
func (l loadGenerator) next(t time.Time) string {
	return NewJSONLogFormat(t)
}

// send sends the event to the publishing pipeline
func (l loadGenerator) send(event string, handle *os.File) error {
	_, err := handle.Write([]byte(event + "\n"))
	return err
}
