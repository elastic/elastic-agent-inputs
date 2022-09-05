// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/elastic-agent-inputs/inputs/loadgenerator"
	"github.com/elastic/elastic-agent-inputs/pkg/outputs/console"
	"github.com/elastic/elastic-agent-inputs/pkg/outputs/shipper"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	defaultConfigName = "agent-inputs.yml"
)

var (
	ConfigPath     string
	ConfigFilePath string
)

// Define some flags used by Elastic-Agent and its friends
func init() {
	fs := flag.CommandLine
	fs.StringVar(&ConfigFilePath, "c", defaultConfigName, "Configuration file, relative to path.config")
	fs.StringVar(&ConfigPath, "path.config", ConfigPath, "Config path is the directory agent-inputs looks for its config file")
}

// Config defines the options present in the config file
type Config struct {
	Log           logp.Config          `yaml:"logging" json:"logging"`
	LoadGenerator loadgenerator.Config `yaml:"loadgenerator" json:"loadgenerator"`
	Outputs       Outputs              `yaml:"outputs"`
}

type Outputs struct {
	Shipper shipper.Config `yaml:"shipper"`
	Console console.Config `yaml:"console"`
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
	logCfg.Beat = "agent-inputs"
	logCfg.ToStderr = true
	logCfg.ToFiles = false

	// While we're in development, keep the default debug
	logCfg.Level = logp.DebugLevel
	logCfg.Selectors = []string{"*"}

	return Config{
		Log:           logCfg,
		LoadGenerator: loadgenerator.DefaultConfig(),
		Outputs: Outputs{
			Shipper: shipper.DefaultConfig(),
		},
	}
}
