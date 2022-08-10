// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/elastic/elastic-agent-inputs/pkg/outputs/console"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher/acker"
	"github.com/elastic/elastic-agent-libs/logp"
)

func Command(logger *logp.Logger, cfg Config) *cobra.Command {
	loadGeneratorCmd := cobra.Command{
		Use:   "loadgenerator",
		Short: "Load generator",
		Run:   run(logger, cfg),
	}

	return &loadGeneratorCmd
}

func run(logger *logp.Logger, cfg Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		logger = logger.Named("loadgenerator")
		ctx := cmd.Context()

		// Create a pipeline, it receives the config necessary to instantiate
		// the output client.
		pipeline := console.NewPipeline(ctx, logger.Named("pipeline"), os.Stdout)

		clientCtx, clientCtxCancel := context.WithCancel(ctx)
		defer clientCtxCancel()

		ackerLogger := logger.Named("acker")
		clientConfig := publisher.ClientConfig{
			PublishMode: publisher.DefaultGuarantees,
			CloseRef:    clientCtx,
			WaitClose:   5 * time.Second,
			ACKHandler:  ackLogger(ackerLogger),
		}

		// Initialises/connects to the output client
		client, err := pipeline.ConnectWith(clientConfig)
		if err != nil {
			logger.Fatalf("could not connect to pipeline: %w", err)
		}

		// initialises the loadGenerator
		lg := loadGenerator{
			cfg:    cfg,
			now:    time.Now,
			logger: logger,
			output: client,
		}

		if err := lg.Run(ctx); err != nil {
			logger.Fatal(err)
		}
	}
}

func ackLogger(logger *logp.Logger) publisher.ACKer {
	fn := func(acked, total int) {
		logger.Debugf("acked: %d, total: %d", acked, total)
	}

	return acker.TrackingCounter(fn)
}
