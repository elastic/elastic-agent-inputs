// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package outputs

import "github.com/elastic/elastic-agent-inputs/pkg/publisher"

type noOp struct{}

func NewNoOp() publisher.PipelineV2 {
	return noOp{}
}

func (n noOp) Publish(publisher.Event)      {}
func (n noOp) PublishAll([]publisher.Event) {}
func (n noOp) Close() error                 { return nil }
