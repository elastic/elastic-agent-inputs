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
	"time"

	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent-inputs/pkg/outputs/console"
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
	logConfig.Beat = "load-generator-input"
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
		Use: "load-generator [subcommand]",
	}
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	// logging flags
	// TODO (Tiago): find out how to properly use them
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))
	// rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	// rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
	// rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))

	runCmd := cobra.Command{
		Use: "run",
		Run: command(logger),
	}
	rootCmd.Run = runCmd.Run

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("oops")
		os.Exit(1)
	}
}

// command is the entry function for the loadGenerator input
// it loads the config, initialises the logger and then
// calls `run` to run the input
func command(logger *logp.Logger) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cfg, err := ReadConfig()
		if err != nil {
			log.Fatal(err)
		}

		if err := logp.Configure(cfg.InputConfig.Log); err != nil {
			panic(fmt.Sprintf("configuring logger: %v", err))
		}
		logger := logp.NewLogger("load-generator-input")

		logger.Debugf("Config: %#v", cfg)
		logger.Info("Starting")

		if err := run(context.TODO(), cfg, logger); err != nil {
			logger.Fatal(err)
		}
	}
}

// run initialises the publishing pipeline, publishing client,
// loadGenerator and starts running the loadGenerator
func run(ctx context.Context, cfg Config, logger *logp.Logger) error {
	// Create a pipeline, it receives the config necessary to instantiate
	// the output client.
	pipeline := console.NewPipeline(ctx, logger.Named("pipeline"), os.Stdout)

	// Initialises/connects to the output client
	client, err := pipeline.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to pipeline: %w", err)
	}

	// initialises the loadGenerator
	lg := loadGenerator{
		cfg:    cfg,
		now:    time.Now,
		logger: logger,
		output: client,
	}

	// runs the loadGenerator until shutdown
	return lg.Run(ctx)
}
