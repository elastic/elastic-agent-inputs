// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package feature

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var defaultDetails = Details{Stability: Stable}

func TestRegister(t *testing.T) {
	f := func() {}

	t.Run("when the factory is nil", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(New("outputs", "null", nil, defaultDetails))
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("namespace and feature doesn't exist", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(New("outputs", "null", f, defaultDetails))
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 1, r.Size())
	})

	t.Run("namespace exists and feature doesn't exist", func(t *testing.T) {
		r := NewRegistry()
		mustRegister(r, New("processor", "bar", f, defaultDetails))
		err := r.Register(New("processor", "foo", f, defaultDetails))
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 2, r.Size())
	})

	t.Run("namespace exists and feature exists and not the same factory", func(t *testing.T) {
		r := NewRegistry()
		mustRegister(r, New("processor", "foo", func() {}, defaultDetails))
		err := r.Register(New("processor", "foo", f, defaultDetails))
		if !assert.Error(t, err) {
			return
		}
		assert.Equal(t, 1, r.Size())
	})

	t.Run("when the exact feature is already registered", func(t *testing.T) {
		feature := New("processor", "foo", f, defaultDetails)
		r := NewRegistry()
		mustRegister(r, feature)
		err := r.Register(feature)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, 1, r.Size())
	})
}

func mustRegister(r *Registry, f Featurable) {
	err := r.Register(f)
	if err != nil {
		panic(err)
	}
}

func TestFeature(t *testing.T) {
	f := func() {}

	r := NewRegistry()
	mustRegister(r, New("processor", "foo", f, defaultDetails))
	mustRegister(r, New("HOLA", "fOO", f, defaultDetails))

	t.Run("when namespace and feature are present", func(t *testing.T) {
		feature, err := r.Lookup("processor", "foo")
		if !assert.NotNil(t, feature.Factory()) {
			return
		}
		assert.NoError(t, err)
	})

	t.Run("when namespace doesn't exist", func(t *testing.T) {
		_, err := r.Lookup("hello", "foo")
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("when namespace and key are normalized", func(t *testing.T) {
		_, err := r.Lookup("HOLA", "foo")
		if !assert.NoError(t, err) {
			return
		}
	})
}

func TestLookup(t *testing.T) {
	f := func() {}

	r := NewRegistry()
	mustRegister(r, New("processor", "foo", f, defaultDetails))
	mustRegister(r, New("processor", "foo2", f, defaultDetails))
	mustRegister(r, New("HELLO", "fOO", f, defaultDetails))

	t.Run("when namespace and feature are present", func(t *testing.T) {
		features, err := r.LookupAll("processor")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, 2, len(features))
	})

	t.Run("when namespace is not present", func(t *testing.T) {
		_, err := r.LookupAll("foobar")
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("when namespace and name are normalized", func(t *testing.T) {
		features, err := r.LookupAll("hello")
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 1, len(features))
	})
}

func TestUnregister(t *testing.T) {
	f := func() {}

	t.Run("when the namespace and the feature exists", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(New("processor", "foo", f, defaultDetails))
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Size())
		err = r.Unregister("processor", "foo")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, 0, r.Size())
	})

	t.Run("when the namespace exist and the feature doesn't", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(New("processor", "foo", f, defaultDetails))
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Size())
		err = r.Unregister("processor", "bar")
		if assert.Error(t, err) {
			return
		}
		assert.Equal(t, 0, r.Size())
	})

	t.Run("when the namespace doesn't exists", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(New("processor", "foo", f, defaultDetails))
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Size())
		err = r.Unregister("outputs", "bar")
		if assert.Error(t, err) {
			return
		}
		assert.Equal(t, 0, r.Size())
	})
}

func TestOverwrite(t *testing.T) {
	t.Run("when the feature doesn't exist", func(t *testing.T) {
		f := func() {}
		r := NewRegistry()
		assert.Equal(t, 0, r.Size())
		err := r.Overwrite(New("processor", "foo", f, defaultDetails))
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Size())
	})

	t.Run("overwrite when the feature exists", func(t *testing.T) {
		f := func() {}
		r := NewRegistry()
		err := r.Register(New("processor", "foo", f, defaultDetails))
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Size())

		check := 42
		err = r.Overwrite(New("processor", "foo", check, defaultDetails))
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Size())

		feature, err := r.Lookup("processor", "foo")
		if !assert.NoError(t, err) {
			return
		}

		v, ok := feature.Factory().(int)
		assert.True(t, ok)
		assert.Equal(t, 42, v)
	})
}
