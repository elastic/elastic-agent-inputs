// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package feature

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundle(t *testing.T) {
	factory := func() {}
	features := []Featurable{
		New("libbeat.outputs", "elasticsearch", factory, Details{Stability: Stable}),
		New("libbeat.outputs", "edge", factory, Details{Stability: Experimental}),
		New("libbeat.input", "tcp", factory, Details{Stability: Beta}),
	}

	t.Run("Creates a new Bundle", func(t *testing.T) {
		b := NewBundle(features...)
		assert.Equal(t, 3, len(b.Features()))
	})

	t.Run("Filters feature based on Stability", func(t *testing.T) {
		b := NewBundle(features...)
		new := b.Filter(Experimental)
		assert.Equal(t, 1, len(new.Features()))
	})

	t.Run("Filters feature based on multiple different Stability", func(t *testing.T) {
		b := NewBundle(features...)
		new := b.Filter(Experimental, Stable)
		assert.Equal(t, 2, len(new.Features()))
	})

	t.Run("Creates a new Bundle from specified feature", func(t *testing.T) {
		f1 := New("libbeat.outputs", "elasticsearch", factory, Details{Stability: Stable})
		b := MustBundle(f1)
		assert.Equal(t, 1, len(b.Features()))
	})

	t.Run("Creates a new Bundle with grouped features", func(t *testing.T) {
		f1 := New("libbeat.outputs", "elasticsearch", factory, Details{Stability: Stable})
		f2 := New("libbeat.outputs", "edge", factory, Details{Stability: Experimental})
		f3 := New("libbeat.input", "tcp", factory, Details{Stability: Beta})
		f4 := New("libbeat.input", "udp", factory, Details{Stability: Beta})

		b := MustBundle(
			MustBundle(f1),
			MustBundle(f2),
			MustBundle(f3, f4),
		)

		assert.Equal(t, 4, len(b.Features()))
	})
}
