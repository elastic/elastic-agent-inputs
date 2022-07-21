// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package memlog

import "github.com/elastic/elastic-agent-libs/mapstr"

type (
	op interface {
		name() string
	}

	// opSet encodes the 'Set' operations in the update log.
	opSet struct {
		K string
		V mapstr.M
	}

	// opRemove encodes the 'Remove' operation in the update log.
	opRemove struct {
		K string
	}
)

// operation type names
const (
	opValSet    = "set"
	opValRemove = "remove"
)

func (*opSet) name() string    { return opValSet }
func (*opRemove) name() string { return opValRemove }
