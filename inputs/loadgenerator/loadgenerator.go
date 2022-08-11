// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

func Command(logger *logp.Logger, cfg Config, output publisher.PipelineV2) *cobra.Command {
	loadGeneratorCmd := cobra.Command{
		Use:   "loadgenerator",
		Short: "Load generator",
		Run:   run(logger, cfg, output),
	}

	return &loadGeneratorCmd
}

func run(logger *logp.Logger, cfg Config, output publisher.PipelineV2) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		logger = logger.Named("loadgenerator")
		ctx := cmd.Context()

		// initialises the loadGenerator
		lg := loadGenerator{
			cfg:    cfg,
			now:    time.Now,
			logger: logger,
			output: output,
		}

		if err := lg.Run(ctx); err != nil {
			logger.Fatal(err)
		}
	}
}
