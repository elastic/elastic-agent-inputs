// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadgenerator

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent-libs/logp"
)

func Command(logger *logp.Logger, cfg Config) *cobra.Command {
	loadGeneratorCmd := cobra.Command{
		Use:   "loadgenerator",
		Short: "Load generator",
		Run:   run(cfg),
	}

	return &loadGeneratorCmd
}

func run(cfg Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		logger := logp.NewLogger("load-generator")
		lg := newLoadGenerator(logger)

		if err := lg.Start(context.TODO()); err != nil {
			logger.Fatal(err)
		}
	}
}
