// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var cnt = 0

func testEvent() publisher.Event {
	event := publisher.Event{
		Fields: mapstr.M{
			"message": "test",
			"idx":     cnt,
		},
	}
	cnt++
	return event
}

// Test that ChanClient writes an event to its Channel.
func TestChanClientPublishEvent(t *testing.T) {
	cc := NewChanClient(1)
	e1 := testEvent()
	cc.Publish(e1)
	assert.Equal(t, e1, cc.ReceiveEvent())
}

// Test that ChanClient write events to its Channel.
func TestChanClientPublishEvents(t *testing.T) {
	cc := NewChanClient(1)

	e1, e2 := testEvent(), testEvent()
	go cc.PublishAll([]publisher.Event{e1, e2})
	assert.Equal(t, e1, cc.ReceiveEvent())
	assert.Equal(t, e2, cc.ReceiveEvent())
}
