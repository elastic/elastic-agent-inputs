// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package input

import (
	"time"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/gofrs/uuid"

	"github.com/elastic/go-concert/unison"
)

// InputManager creates and maintains actions and background processes for an
// input type.
// The InputManager is used to create inputs. The InputManager can provide
// additional functionality like coordination between input of the same type,
// custom functionality for querying or caching shared information, application
// of common settings not unique to a particular input type, or require a more
// specific Input interface to be implemented by the actual input.
type InputManager interface {
	// Init signals to InputManager to initialize internal resources.
	// The mode tells the input manager if the Beat is actually running the inputs or
	// if inputs are only configured for testing/validation purposes.
	Init(grp unison.Group, mode Mode) error

	// Creates builds a new Input instance from the given configuation, or returns
	// an error if the configuation is invalid.
	// The input must establish any connection for data collection yet. The Beat
	// will use the Test/Run methods of the input.
	Create(*conf.C) (Input, error)
}

// Mode tells the InputManager in which mode it is initialized.
type Mode uint8

//go:generate stringer -type Mode -trimprefix Mode
const (
	ModeRun Mode = iota
	ModeTest
	ModeOther
)

// Input is a configured input object that can be used to test or start
// the actual data collection.
type Input interface {
	// Name reports the input name.
	//
	// XXX: check if/how we can remove this method. Currently it is required for
	// compatibility reasons with existing interfaces in libbeat, autodiscovery
	// and filebeat.
	Name() string

	// Test checks the configuaration and runs additional checks if the Input can
	// actually collect data for the given configuration (e.g. check if host/port or files are
	// accessible).
	Test(TestContext) error

	// Run starts the data collection. Run must return an error only if the
	// error is fatal making it impossible for the input to recover.
	Run(Context, publisher.PipelineConnector) error
}

// Info stores a input instance meta data.
type Info struct {
	Input       string    // The actual beat's name
	IndexPrefix string    // The beat's index prefix in Elasticsearch.
	Version     string    // The beat version. Defaults to the libbeat version when an implementation does not set a version
	Name        string    // configured beat name
	Hostname    string    // hostname
	ID          uuid.UUID // ID assigned to beat machine
	EphemeralID uuid.UUID // ID assigned to beat process invocation (PID)
	FirstStart  time.Time // The time of the first start of the Beat.
	StartTime   time.Time // The time of last start of the Beat. Updated when the Beat is started or restarted.

	// Monitoring-related fields
	Monitoring struct {
		DefaultUsername string // The default username to be used to connect to Elasticsearch Monitoring
	}
}

// Context provides the Input Run function with common environmental
// information and services.
type Context struct {
	// Logger provides a structured logger to inputs. The logger is initialized
	// with labels that will identify logs for the input.
	Logger *logp.Logger

	// The input ID.
	ID string

	// Agent provides additional Beat info like instance ID or beat name.
	Agent Info

	// Cancelation is used by Beats to signal the input to shutdown.
	Cancelation Canceler
}

// TestContext provides the Input Test function with common environmental
// information and services.
type TestContext struct {
	// Logger provides a structured logger to inputs. The logger is initialized
	// with labels that will identify logs for the input.
	Logger *logp.Logger

	// Agent provides additional info like instance ID or beat name.
	Agent Info

	// Cancelation is used by the binary to signal the input to shutdown.
	Cancelation Canceler
}

// Canceler is used to provide shutdown handling to the Context.
type Canceler interface {
	Done() <-chan struct{}
	Err() error
}
