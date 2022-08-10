// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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

	// Configure the logp package
	if err := logp.Configure(cfg.Log); err != nil {
		logger.Fatalf("applying logger configuration: %v", err)
	}

	// Get a new logger with the configuration we've just applied
	logger = logp.L()

	// Set the output of the standard logger to our new logger.
	log.SetOutput(logWriter{logger})

	rootCmd.AddCommand(loadgenerator.Command(logger, cfg.LoadGenerator))

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Fatal(err)
	}
}
