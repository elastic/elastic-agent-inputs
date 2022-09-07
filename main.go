// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-inputs/inputs/loadgenerator"
	"github.com/elastic/elastic-agent-inputs/pkg/config"
	"github.com/elastic/elastic-agent-inputs/pkg/outputs"
	"github.com/elastic/elastic-agent-inputs/pkg/outputs/console"
	"github.com/elastic/elastic-agent-inputs/pkg/outputs/shipper"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher/acker"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher/pipeline"
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

	output, err := initPublishingPipeline(ctx, cfg, logger)
	if err != nil {
		logger.Fatalf("could not initialise publishing pipeline: %s", err)
	}

	agentAddr := net.JoinHostPort(cfg.ElasticAgent.Host, strconv.Itoa(cfg.ElasticAgent.Port))
	logger.Infof("connecting to Elastic-Agent on: '%s'", agentAddr)

	client := client.NewV2(agentAddr, cfg.ElasticAgent.Token, client.VersionInfo{
		Name:    "elastic-agent-inputs",
		Version: "v1.0.0",
	}, grpc.WithTransportCredentials(insecure.NewCredentials()))

	rootCmd.AddCommand(loadgenerator.Command(logger, cfg.LoadGenerator, client, output))

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Fatal(err)
	}
}

func initPublishingPipeline(ctx context.Context, cfg config.Config, logger *logp.Logger) (publisher.PipelineV2, error) {
	// 1. Initialise ackerInstance
	ackerInstance := acker.NoOp()

	// 2. Initialise output
	output := initOutput(ctx, cfg.Outputs, ackerInstance, logger)

	// 3. Initialise publishing pipeline
	pipeline := pipeline.New(
		ctx,
		logger.Named("pipeline"),
		output,
		nil, // processors list
	)

	return pipeline, nil
}

func initOutput(ctx context.Context, cfg config.Outputs, ackerInstance publisher.ACKer, logger *logp.Logger) publisher.PipelineV2 {
	var client publisher.PipelineV2
	var err error

	switch {
	case cfg.Console.Enabled:
		client = console.New(
			ctx,
			os.Stdout,
			logger.Named("console_client"),
			ackerInstance,
			cfg.Console,
		)
	case cfg.Shipper.Enabled:
		client, err = shipper.New(
			ctx,
			cfg.Shipper,
			logger.Named("shipper_client"),
			ackerInstance,
		)
		if err != nil {
			logger.Fatalf("could not initialise shipper: %s", err)
		}

	default:
		logger.Warn("no output enabled, using a no-op output")
		client = outputs.NewNoOp()
	}

	return client
}

func ackLogger(logger *logp.Logger) publisher.ACKer {
	fn := func(acked, total int) {
		logger.Debugf("acked: %d, total: %d", acked, total)
	}

	return acker.TrackingCounter(fn)
}
