// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"google.golang.org/protobuf/types/known/structpb"
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

	Port  int    `yaml:"port" json:"port"`
	Token string `yaml:"token" json:"token"`
}

func DefaultConfig() Config {
	return Config{
		Loop:        true,
		Delay:       time.Second,
		TimeDelta:   time.Second,
		CurrentTime: false,
	}
}

// configFromSource creates the config struct from the protobuf source map
func configFromSource(source *structpb.Struct) (Config, error) {
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
