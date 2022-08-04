// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	intConfig "github.com/elastic/elastic-agent-inputs/pkg/config"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	defaultConfigName = "load-generator.yml"
)

var (
	ConfigPath     string
	ConfigFilePath string
)

// Define some flags used by Elastic-Agent and its friends
func init() {
	fs := flag.CommandLine
	fs.StringVar(&ConfigFilePath, "c", defaultConfigName, "Configuration file, relative to path.config")
	fs.StringVar(&ConfigPath, "path.config", ConfigPath, "Config path is the directory Inputs looks for its config file")
}

type Config struct {
	// InputConfig is the common input configuration
	InputConfig intConfig.Config `config:",inline"`

	// Delay between generating events, it does not affect their timestamp.
	// Use TimeDelta to set the timestamp increments
	Delay time.Duration `yaml:"delay" json:"delay"`

	// TimeDelta between events
	// TODO (Tiago): find out why "time_delta" does not work
	TimeDelta time.Duration `yaml:"timedelta" json:"timepdelta"`

	// CurrentTime uses the current time when generating events instead of TimeDelta
	CurrentTime bool `yaml:"currenttime" json:"currenttime"`

	// EventsCount sets the number of events to generate, if Loop is set
	// to true, this field is ignored.
	EventsCount uint64 `yaml:"eventscount" json:"eventscount"`

	// Loop forever until the process is shutdown
	Loop bool `yaml:"loop" json:"loop"`
}

// ReadConfig returns the populated config from the specified path
func ReadConfig() (Config, error) {
	path := Filepath()

	configYAML, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("error reading input file '%s': %w", path, err)
	}

	raw, err := config.NewConfigWithYAML(configYAML, "")
	if err != nil {
		return Config{}, fmt.Errorf("error parsing yaml config: %w", err)
	}

	config := defaultConfig()
	if err := raw.Unpack(&config); err != nil {
		return config, fmt.Errorf("error unpacking shipper config: %w", err)
	}

	return config, nil
}

// Filepath returns the default config file path or the config
// file path set as flags
func Filepath() string {
	if ConfigFilePath == "" || ConfigFilePath == defaultConfigName {
		return filepath.Join(ConfigPath, defaultConfigName)
	}
	if filepath.IsAbs(ConfigFilePath) {
		return ConfigFilePath
	}
	return filepath.Join(ConfigPath, ConfigFilePath)
}

func defaultConfig() Config {
	logCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	logCfg.Beat = "load-generator-input"
	logCfg.ToStderr = true
	logCfg.ToFiles = false
	// While we're in development, keep the default debug
	// TODO (Tiago): set it to Info before releasing as beta
	logCfg.Level = logp.DebugLevel
	logCfg.Selectors = []string{"*"}

	return Config{
		InputConfig: intConfig.Config{
			Log: logCfg,
		},
		Loop:        true,
		Delay:       time.Second,
		TimeDelta:   time.Second,
		CurrentTime: false,
	}
}
