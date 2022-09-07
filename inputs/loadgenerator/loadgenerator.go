// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-inputs/pkg/manager/input"
	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

func Command(logger *logp.Logger, cfg Config, agentClient client.V2, output publisher.PipelineV2) *cobra.Command {
	loadGeneratorCmd := cobra.Command{
		Use:   "loadgenerator",
		Short: "Load generator",
		Run:   run(logger, cfg, agentClient, output),
	}

	return &loadGeneratorCmd
}

func run(logger *logp.Logger, cfg Config, agentClient client.V2, output publisher.PipelineV2) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		logger = logger.Named("loadgenerator")
		ctx := cmd.Context()

		spkm := input.NewSpikeManager(logger.Named("spike-manager"), agentClient, output, NewLoadGenerator)
		spkm.Init(ctx)
	}
}
