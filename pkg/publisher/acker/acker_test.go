// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package acker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
)

type fakeACKer struct {
	AddEventFunc  func(event publisher.Event, published bool)
	ACKEventsFunc func(n int)
	CloseFunc     func()
}

func TestNil(t *testing.T) {
	acker := Nil()
	require.NotNil(t, acker)

	// check acker can be used without panic:
	acker.AddEvent(publisher.Event{}, false)
	acker.AddEvent(publisher.Event{}, true)
	acker.ACKEvents(3)
	acker.Close()
}

func TestCounting(t *testing.T) {
	t.Run("ack count is passed through", func(t *testing.T) {
		var n int
		acker := RawCounting(func(acked int) { n = acked })
		acker.ACKEvents(3)
		require.Equal(t, 3, n)
	})
}

func TestTracking(t *testing.T) {
	t.Run("dropped event is acked immediately if empty", func(t *testing.T) {
		var acked, total int
		TrackingCounter(func(a, t int) { acked, total = a, t }).AddEvent(publisher.Event{}, false)
		require.Equal(t, 0, acked)
		require.Equal(t, 1, total)
	})

	t.Run("no dropped events", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 2, total)
	})

	t.Run("acking published includes dropped events in middle", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, false)
		acker.AddEvent(publisher.Event{}, false)
		acker.AddEvent(publisher.Event{}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 4, total)
	})

	t.Run("acking published includes dropped events at end of ACK interval", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, false)
		acker.AddEvent(publisher.Event{}, false)
		acker.AddEvent(publisher.Event{}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 4, total)
	})

	t.Run("partial ACKs", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, false)
		acker.AddEvent(publisher.Event{}, true)
		acker.AddEvent(publisher.Event{}, true)

		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 2, total)

		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 3, total)
	})
}

//nolint:dupl //tests are similar, not duplicated
func TestEventPrivateReporter(t *testing.T) {
	t.Run("dropped event is acked immediately if empty", func(t *testing.T) {
		var acked int
		var data []interface{}
		acker := EventPrivateReporter(func(a int, d []interface{}) { acked, data = a, d })
		acker.AddEvent(publisher.Event{Private: 1}, false)
		require.Equal(t, 0, acked)
		require.Equal(t, []interface{}{1}, data)
	})

	t.Run("no dropped events", func(t *testing.T) {
		var acked int
		var data []interface{}
		acker := EventPrivateReporter(func(a int, d []interface{}) { acked, data = a, d })
		acker.AddEvent(publisher.Event{Private: 1}, true)
		acker.AddEvent(publisher.Event{Private: 2}, true)
		acker.AddEvent(publisher.Event{Private: 3}, true)
		acker.ACKEvents(3)
		require.Equal(t, 3, acked)
		require.Equal(t, []interface{}{1, 2, 3}, data)
	})

	t.Run("private of dropped events is included", func(t *testing.T) {
		var acked int
		var data []interface{}
		acker := EventPrivateReporter(func(a int, d []interface{}) { acked, data = a, d })
		acker.AddEvent(publisher.Event{Private: 1}, true)
		acker.AddEvent(publisher.Event{Private: 2}, false)
		acker.AddEvent(publisher.Event{Private: 3}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, []interface{}{1, 2, 3}, data)
	})
}

func TestLastEventPrivateReporter(t *testing.T) {
	t.Run("dropped event with private is acked immediately if empty", func(t *testing.T) {
		var acked int
		var datum interface{}
		acker := LastEventPrivateReporter(func(a int, d interface{}) { acked, datum = a, d })
		acker.AddEvent(publisher.Event{Private: 1}, false)
		require.Equal(t, 0, acked)
		require.Equal(t, 1, datum)
	})

	t.Run("dropped event without private is ignored", func(t *testing.T) {
		var called bool
		acker := LastEventPrivateReporter(func(_ int, _ interface{}) { called = true })
		acker.AddEvent(publisher.Event{Private: nil}, false)
		require.False(t, called)
	})

	t.Run("no dropped events", func(t *testing.T) {
		var acked int
		var data interface{}
		acker := LastEventPrivateReporter(func(a int, d interface{}) { acked, data = a, d })
		acker.AddEvent(publisher.Event{Private: 1}, true)
		acker.AddEvent(publisher.Event{Private: 2}, true)
		acker.AddEvent(publisher.Event{Private: 3}, true)
		acker.ACKEvents(3)
		require.Equal(t, 3, acked)
		require.Equal(t, 3, data)
	})
}

func TestCombine(t *testing.T) {
	t.Run("AddEvent distributes", func(t *testing.T) {
		var a1, a2 int
		acker := Combine(countACKerOps(&a1, nil, nil), countACKerOps(&a2, nil, nil))
		acker.AddEvent(publisher.Event{}, true)
		require.Equal(t, 1, a1)
		require.Equal(t, 1, a2)
	})

	t.Run("ACKEvents distributes", func(t *testing.T) {
		var a1, a2 int
		acker := Combine(countACKerOps(nil, &a1, nil), countACKerOps(nil, &a2, nil))
		acker.ACKEvents(1)
		require.Equal(t, 1, a1)
		require.Equal(t, 1, a2)
	})

	t.Run("Close distributes", func(t *testing.T) {
		var c1, c2 int
		acker := Combine(countACKerOps(nil, nil, &c1), countACKerOps(nil, nil, &c2))
		acker.Close()
		require.Equal(t, 1, c1)
		require.Equal(t, 1, c2)
	})
}

func TestConnectionOnly(t *testing.T) {
	t.Run("passes ACKs if not closed", func(t *testing.T) {
		var n int
		acker := ConnectionOnly(RawCounting(func(acked int) { n = acked }))
		acker.ACKEvents(3)
		require.Equal(t, 3, n)
	})

	t.Run("ignores ACKs after close", func(t *testing.T) {
		var n int
		acker := ConnectionOnly(RawCounting(func(acked int) { n = acked }))
		acker.Close()
		acker.ACKEvents(3)
		require.Equal(t, 0, n)
	})
}

func countACKerOps(add, acked, close *int) publisher.ACKer {
	return &fakeACKer{
		AddEventFunc:  func(_ publisher.Event, _ bool) { *add++ },
		ACKEventsFunc: func(_ int) { *acked++ },
		CloseFunc:     func() { *close++ },
	}
}

func (f *fakeACKer) AddEvent(event publisher.Event, published bool) {
	if f.AddEventFunc != nil {
		f.AddEventFunc(event, published)
	}
}

func (f *fakeACKer) ACKEvents(n int) {
	if f.ACKEventsFunc != nil {
		f.ACKEventsFunc(n)
	}
}

func (f *fakeACKer) Close() {
	if f.CloseFunc != nil {
		f.CloseFunc()
	}
}
