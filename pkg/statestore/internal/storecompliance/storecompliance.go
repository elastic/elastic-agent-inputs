// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package storecompliance provides a common test suite that a store
// implementation must succeed in order to be compliant to the beats
// statestore. The Internal tests are used by statestore/storetest and
// statestore/backend/memlog.
//
// The package adds the `-keep` and `-dir <path>` CLI flags:
//   - `-dir <path>`: configure path where to create test folders in (defaults
//     to OS specific temporary directory)
//   - `-keep`: The test directories will not be deleted after a test has
//     finished. The test directory is added to the test logs.
//
package storecompliance

import (
	"errors"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-inputs/pkg/statestore/backend"
)

// BackendFactory is used by TestBackendCompliance to create
// store instances for testing. Each store will be configured
// with an unique temporary directory.
type BackendFactory func(testPath string) (backend.Registry, error)

var defaultTempDir string
var keepTmpDir bool

func init() {
	flag.StringVar(&defaultTempDir, "dir", "", "Temporary directory for use by the test")
	flag.BoolVar(&keepTmpDir, "keep", false, "Keep temporary test directories")
}

// TestBackendCompliance runs a set of tests the verifies that the store
// implementation can be used by beats.
// Most tests are executed twice if they modify data. Once with keeping the
// store open between operations, and once with reopening the store between
// updates.
// For a store backend that supports different 'modes' that can impact storage,
// the compliance tests should be run with the different modes enabled.
//
// Note: The tests only check for interoperability. Implementations should add
// additional tests as well.
func TestBackendCompliance(t *testing.T, factory BackendFactory) {
	t.Run("init and close registry", WithPath(factory, func(t *testing.T, reg *Registry) {
	}))

	t.Run("open stores", WithPath(factory, func(t *testing.T, reg *Registry) {
		store := reg.MustAccess("test1")
		defer store.Close()

		store2 := reg.MustAccess("test2")
		defer store2.Close()
	}))

	t.Run("set-get", withBackend(factory, testSetGet))
	t.Run("remove", withBackend(factory, testRemove))
	t.Run("iteration", withBackend(factory, testIteration))
}

func testSetGet(t *testing.T, factory BackendFactory) {
	t.Run("unknown key", WithStore(factory, func(t *testing.T, store *Store) {
		has := store.MustHave("key")
		assert.False(t, has)
	}))

	runWithBools(t, "reopen", func(t *testing.T, reopen bool) {
		t.Run("has key after set", WithStore(factory, func(t *testing.T, store *Store) {
			type entry struct{ A int }
			store.MustSet("key", entry{A: 1})

			store.ReopenIf(reopen)

			has := store.MustHave("key")
			assert.True(t, has)
		}))

		t.Run("set and get one entry only", WithStore(factory, func(t *testing.T, store *Store) {
			type entry struct{ A int }
			key := "key"
			value := entry{A: 1}

			store.MustSet(key, value)
			store.ReopenIf(reopen)

			var actual entry
			store.MustGet(key, &actual)
			assert.Equal(t, value, actual)
		}))
	})
}

func testRemove(t *testing.T, factory BackendFactory) {
	t.Run("no error when removing unknown key", WithStore(factory, func(t *testing.T, store *Store) {
		store.MustRemove("key")
	}))

	runWithBools(t, "reopen", func(t *testing.T, reopen bool) {
		t.Run("remove key", WithStore(factory, func(t *testing.T, store *Store) {
			type entry struct{ A int }
			key := "key"
			store.MustSet(key, entry{A: 1})
			store.ReopenIf(reopen)
			store.MustRemove(key)
			store.ReopenIf(reopen)
			has := store.MustHave(key)
			assert.False(t, has)
		}))
	})
}

func testIteration(t *testing.T, factory BackendFactory) {
	data := map[string]interface{}{
		"a": map[string]interface{}{"field": "hello"},
		"b": map[string]interface{}{"field": "world"},
	}

	addTestData := func(store *Store, reopen bool, data map[string]interface{}) {
		for k, v := range data {
			store.MustSet(k, v)
		}
		store.ReopenIf(reopen)
	}

	runWithBools(t, "reopen", func(t *testing.T, reopen bool) {
		t.Run("all keys", WithStore(factory, func(t *testing.T, store *Store) {
			addTestData(store, reopen, data)

			got := map[string]interface{}{}
			err := store.Each(func(key string, dec backend.ValueDecoder) (bool, error) {
				var tmp interface{}
				if err := dec.Decode(&tmp); err != nil {
					return false, err
				}

				got[key] = tmp
				return true, nil
			})

			assert.NoError(t, err)
			assert.Equal(t, data, got)
		}))

		t.Run("stop on error", WithStore(factory, func(t *testing.T, store *Store) {
			addTestData(store, reopen, data)

			count := 0
			err := store.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
				count++
				return true, errors.New("oops")
			})
			assert.Equal(t, 1, count)
			assert.Error(t, err)
		}))

		t.Run("stop on bool", WithStore(factory, func(t *testing.T, store *Store) {
			addTestData(store, reopen, data)

			count := 0
			err := store.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
				count++
				return false, nil
			})
			assert.Equal(t, 1, count)
			assert.NoError(t, err)
		}))
	})
}
