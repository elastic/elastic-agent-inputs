// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package statestore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessStore(t *testing.T) {
	t.Run("single access", func(t *testing.T) {
		mr := newMockRegistry()
		ms := newMockStore()
		mr.OnClose().Once().Return(nil)
		mr.OnAccess("test").Once().Return(ms, nil)
		ms.OnClose().Once().Return(nil)

		reg := NewRegistry(mr)
		store, _ := reg.Get("test")
		assert.NoError(t, store.Close())
		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("shared store instance", func(t *testing.T) {
		mr := newMockRegistry()
		ms := newMockStore()
		mr.OnClose().Once().Return(nil)

		// test instance sharing. Store must be opened and closed only once
		mr.OnAccess("test").Once().Return(ms, nil)
		ms.OnClose().Once().Return(nil)

		reg := NewRegistry(mr)
		s1, _ := reg.Get("test")
		s2, _ := reg.Get("test")
		assert.NoError(t, s1.Close())
		assert.NoError(t, s2.Close())
		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("close non-shared store needs open", func(t *testing.T) {
		mr := newMockRegistry()
		ms := newMockStore()
		mr.OnClose().Once().Return(nil)

		// test instance sharing. Store must be opened and closed only once
		mr.OnAccess("test").Twice().Return(ms, nil)
		ms.OnClose().Twice().Return(nil)

		reg := NewRegistry(mr)

		store, err := reg.Get("test")
		assert.NoError(t, err)
		assert.NoError(t, store.Close())

		store, err = reg.Get("test")
		assert.NoError(t, err)
		assert.NoError(t, store.Close())

		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("separate stores are not shared", func(t *testing.T) {
		mr := newMockRegistry()
		mr.OnClose().Once().Return(nil)

		ms1 := newMockStore()
		ms1.OnClose().Once().Return(nil)
		mr.OnAccess("s1").Once().Return(ms1, nil)

		ms2 := newMockStore()
		ms2.OnClose().Once().Return(nil)
		mr.OnAccess("s2").Once().Return(ms2, nil)

		reg := NewRegistry(mr)
		s1, err := reg.Get("s1")
		assert.NoError(t, err)
		s2, err := reg.Get("s2")
		assert.NoError(t, err)
		assert.NoError(t, s1.Close())
		assert.NoError(t, s2.Close())
		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms1.AssertExpectations(t)
		ms2.AssertExpectations(t)
	})
}
