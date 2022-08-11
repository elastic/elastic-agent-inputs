// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent-inputs/inputs/loadgenerator"
	"github.com/elastic/elastic-agent-inputs/pkg/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type logWriter struct {
	l *logp.Logger
}

func (l logWriter) Write(p []byte) (n int, err error) {
	l.l.Info(string(p))
	return len(p), nil
}

func main() {
	// Initialise the logger as early as possible
	logConfig := logp.DefaultConfig(logp.DefaultEnvironment)
	logConfig.Beat = "agent-inputs"
	logConfig.ToStderr = true
	logConfig.ToFiles = false

	if err := logp.Configure(logConfig); err != nil {
		panic(fmt.Errorf("could not initialise logger: %w", err))
	}
	logger := logp.L()

	// Sets the output destination for the standard logger, if any package uses
	// the std lib logger, we get a nice JSON output
	log.SetOutput(logWriter{logger})

	rootCmd := &cobra.Command{
		Use: "agent-inputs",
	}
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))

	cfg, err := config.ReadConfig()
	if err != nil {
		logger.Fatalf("could not read config file: %s", err)
	}

	if err := logp.Configure(cfg.Log); err != nil {
		logger.Fatalf("applying logger configuration: %v", err)
	}

	rootCmd.AddCommand(loadgenerator.Command(logger, cfg.LoadGenerator))

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}
