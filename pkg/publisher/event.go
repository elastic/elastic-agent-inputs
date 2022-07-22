// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package publisher

import "github.com/elastic/elastic-agent-libs/mapstr"

type Event struct {
	Fields  mapstr.M
	Private interface{}
}
