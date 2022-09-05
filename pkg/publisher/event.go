// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package publisher

import (
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

// Event represents an event that can be run through the processors and sent
// to the Shipper
type Event struct {
	// ShipperMessage is the data type the Sipper will accept to publish
	ShipperMessage *messages.Event

	// Private is private data that will not be published
	Private interface{}
}
