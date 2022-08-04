// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"github.com/elastic/elastic-agent-libs/logp"

	// This adds the global log flags
	_ "github.com/elastic/elastic-agent-libs/logp/configure"
)

// TODO (Tiago): make those global, at the moment they're defined
// in the load-generator input
// var (
// 	ConfigPath     string
// 	ConfigFilePath string
// )

// // Define some flags used by Elastic-Agent and its friends
// func init() {
// 	fs := flag.CommandLine
// 	fs.StringVar(&ConfigFilePath, "c", defaultConfigName, "Configuration file, relative to path.config")
// 	fs.StringVar(&ConfigPath, "path.config", ConfigPath, "Config path is the directory Inputs looks for its config file")
// }

// Config defines the options present in the config file
// TODO (Tiago): fix config parsing/unpacking
type Config struct {
	Log logp.Config `yaml:"logging" json:"logging"`
}
