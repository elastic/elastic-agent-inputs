// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"time"
)

type Config struct {
	// Delay between generating events, it does not affect their timestamp.
	// Use TimeDelta to set the timestamp increments
	Delay time.Duration `yaml:"delay" json:"delay"`

	// TimeDelta between events
	TimeDelta time.Duration `yaml:"timedelta" json:"timepdelta"`

	// CurrentTime uses the current time when generating events instead of TimeDelta
	CurrentTime bool `yaml:"currenttime" json:"currenttime"`

	// EventsCount sets the number of events to generate, if Loop is set
	// to true, this field is ignored.
	EventsCount uint64 `yaml:"eventscount" json:"eventscount"`

	// Loop forever until the process is shutdown
	Loop bool `yaml:"loop" json:"loop"`

	// Send the generated output to a file
	// Eventually we want to send this to the shipper, but for now a file works
	// If this is unset, default to stdout
	OutFile string `yaml:"outfile" json:"outfile"`
}

func DefaultConfig() Config {
	return Config{
		Loop:        true,
		Delay:       time.Second,
		TimeDelta:   time.Second,
		CurrentTime: false,
	}
}
