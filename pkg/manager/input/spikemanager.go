package input

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/types/known/structpb"
)

type InputV2 interface {
	// Run runs the input and blocks until the input has finished
	Run(stopChan chan struct{}) error

	// Stop stops the input, the context is used for timeout
	// it blocks until the input has closed
	Stop(ctx context.Context) error
}

type CreateInputFunc func(logger *logp.Logger, output publisher.PipelineV2, source *structpb.Struct, unit *client.Unit, streamID string) (InputV2, error)

type SpikeManager struct {
	agentClient client.V2
	// configure   func(unit *client.Unit) (Input, error)
	logger   *logp.Logger
	output   publisher.Client
	inputs   map[string]InputV2
	createFn CreateInputFunc

	runners map[string]*loadRunner
	runMut  *sync.Mutex
	events  atomic.Uint64
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

// NewSpikeManager returns a new SpikeManager that can manage inputs.
// The context is used for cancelation
func NewSpikeManager(
	logger *logp.Logger,
	agentClient client.V2,
	output publisher.Client,
	createFn CreateInputFunc,
) SpikeManager {

	return SpikeManager{
		logger:      logger,
		agentClient: agentClient,
		output:      output,
		createFn:    createFn,

		runMut:  &sync.Mutex{},
		runners: map[string]*loadRunner{},
	}
}

// Init initialises everything that ban be shared between input instances
// TODO (Tiago): Maybe this should receive a client.Unit ?
func (s *SpikeManager) Init(ctx context.Context) error {
	if err := s.agentClient.Start(ctx); err != nil {
		return fmt.Errorf("SpikeManager cannot start Elastic-Agent client: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("got context done")
			return nil
		case change := <-s.agentClient.UnitChanges():
			switch change.Type {
			case client.UnitChangedAdded:
				s.logger.Infof("got new Unit with ID: '%s'", change.Unit.ID())
				err := s.handleNewUnit(change.Unit)
				if err != nil {
					_ = change.Unit.UpdateState(client.UnitStateFailed, fmt.Sprintf("New unit failed with: %s", err), nil)
				}

			case client.UnitChangedRemoved:
				s.logger.Debugf("Got unit removed for ID %s", change.Unit.ID())
				s.removeUnit(change.Unit)
				if s.checkDone() {
					s.logger.Debugf("All active units removed, exiting")
					s.logger.Infof("Wrote a total of %d Events", s.events.Load())
					return nil
				}

			case client.UnitChangedModified:
				s.logger.Debugf("Got unit modified")
				s.handleUnitModified(change.Unit)
			}
		}
	}
}

// Create creates a new input instance based on the given Unit
func (s *SpikeManager) Create(unit *client.Unit) (Input, error) {
	return nil, errors.New("not implemented")
}

// Modify updates the input corresponding to the unit and returns
// the updated instance
func (s *SpikeManager) Modify(unit *client.Unit) (Input, error) {
	return nil, errors.New("Not implemeted")
}

// Remove removes/stops an input
func (s *SpikeManager) Remove(unit *client.Unit) {}

// handleNewUnit starts a new unit and initializes a collection of runners from the config
func (s *SpikeManager) handleNewUnit(unit *client.Unit) error {
	s.runMut.Lock()
	defer s.runMut.Unlock()
	// As far as configuration: each stream will be treated as a different load-generator run,
	// This way the config can easily scale up the number of parallel runs.
	_, _, cfg := unit.Expected()
	mgr := newLoadRunner()
	_ = unit.UpdateState(client.UnitStateConfiguring, "configuring unit", nil)
	for _, runner := range cfg.Streams {
		streamID := runner.Id

		lgen, err := s.createFn(s.logger.Named("loadgenerator"), s.output, runner.Source, unit, streamID)
		if err != nil {
			return fmt.Errorf("could not create input: %w", err)
		}

		stop := make(chan struct{}, 1)
		_ = unit.UpdateState(client.UnitStateStarting, fmt.Sprintf("Starting stream with ID %s", streamID), nil)
		go func() {
			err = lgen.Run(stop)
			if err != nil {
				_ = unit.UpdateState(client.UnitStateDegraded, fmt.Sprintf("Error running stream with ID %s: %s", streamID, err), nil)
			}
		}()

		mgr.stopFuncs[streamID] = stop
	}

	mgr.unit = unit
	s.runners[unit.ID()] = mgr
	_ = unit.UpdateState(client.UnitStateHealthy, "started runners", nil)
	return nil
}

// handleUnitModified wraps any functions needed to manage changes to a unit
func (s *SpikeManager) handleUnitModified(unit *client.Unit) {
	state, _, _ := unit.Expected()
	if state == client.UnitStateStopping || state == client.UnitStateStopped {
		s.tearDownUnit(unit)
	} else {
		// Not sure how this should respond to unit changes.
		// Reload the entire unit? Diff all the streams for changes, and reload individual streams?
		s.logger.Debugf("Got unit state: %s", state)
	}
}

// tearDownUnit stops a unit, and all streams associated with it
func (s *SpikeManager) tearDownUnit(unit *client.Unit) {
	s.runMut.Lock()
	defer s.runMut.Unlock()

	_ = unit.UpdateState(client.UnitStateStopping, "starting unit shutdown", nil)

	id := unit.ID()
	unitRunner := s.runners[id]
	unitRunner.mapMut.Lock()
	for streamID, runner := range unitRunner.stopFuncs {
		s.logger.Debugf("Stopping runner %s", streamID)
		_ = unit.UpdateState(client.UnitStateStopping, fmt.Sprintf("sending shutdown to unit %s", streamID), nil)
		runner <- struct{}{}
		// remove the stream from our map
		delete(unitRunner.stopFuncs, streamID)
	}
	unitRunner.mapMut.Unlock()

	s.logger.Debugf("All runners for unit %s have stopped", id)
	_ = unit.UpdateState(client.UnitStateStopped, "all runners for unit have stopped", nil)
}

// removeUnit deletes a unit from the map. This should be called after we've removed any streams associated with a unit.
func (s *SpikeManager) removeUnit(unit *client.Unit) {
	s.runMut.Lock()
	defer s.runMut.Unlock()
	delete(s.runners, unit.ID())
}

// checkDone checks to see if we have any running units.
func (s *SpikeManager) checkDone() bool {
	s.runMut.Lock()
	defer s.runMut.Unlock()
	return len(s.runners) == 0
}
