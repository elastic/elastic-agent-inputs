// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storetest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-inputs/pkg/statestore/backend"
	"github.com/elastic/elastic-agent-inputs/pkg/statestore/internal/storecompliance"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	err := logp.DevelopmentSetup()
	if err != nil {
		panic(err)
	}
}

func TestCompliance(t *testing.T) {
	storecompliance.TestBackendCompliance(t, func(testPath string) (backend.Registry, error) {
		return NewMemoryStoreBackend(), nil
	})
}

func TestStore_IsClosed(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		store := &MapStore{}
		assert.False(t, store.IsClosed())
	})
	t.Run("true after close", func(t *testing.T) {
		store := &MapStore{}
		store.Close()
		assert.True(t, store.IsClosed())
	})
	t.Run("true after reopen", func(t *testing.T) {
		store := &MapStore{}
		store.Close()
		store.Reopen()
		assert.False(t, store.IsClosed())
	})
}
