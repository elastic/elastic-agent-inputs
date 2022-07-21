// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestGCStore(t *testing.T) {
	t.Run("empty store", func(t *testing.T) {
		started := time.Now()

		backend := createSampleStore(t, nil)
		store := testOpenStore(t, backend)
		defer store.Release()

		gcStore(logp.NewLogger("test"), started, store)

		want := map[string]state{}
		checkEqualStoreState(t, want, backend.snapshot())
	})

	t.Run("state is still alive", func(t *testing.T) {
		started := time.Now()
		const ttl = 60 * time.Second

		initState := map[string]state{
			"test::key": {
				TTL:     ttl,
				Updated: started.Add(-ttl / 2),
			},
		}

		backend := createSampleStore(t, initState)
		store := testOpenStore(t, backend)
		defer store.Release()

		gcStore(logp.NewLogger("test"), started, store)

		checkEqualStoreState(t, initState, backend.snapshot())
	})

	t.Run("old state can be removed", func(t *testing.T) {
		const ttl = 60 * time.Second
		started := time.Now().Add(-5 * ttl) // cleanup process is running for a while already

		initState := map[string]state{
			"test::key": {
				TTL:     ttl,
				Updated: started.Add(-ttl),
			},
		}

		backend := createSampleStore(t, initState)
		store := testOpenStore(t, backend)
		defer store.Release()

		gcStore(logp.NewLogger("test"), started, store)

		want := map[string]state{}
		checkEqualStoreState(t, want, backend.snapshot())
	})

	t.Run("old state is not removed if cleanup is not active long enough", func(t *testing.T) {
		const ttl = 60 * time.Minute
		started := time.Now()

		initState := map[string]state{
			"test::key": {
				TTL:     ttl,
				Updated: started.Add(-2 * ttl),
			},
		}

		backend := createSampleStore(t, initState)
		store := testOpenStore(t, backend)
		defer store.Release()

		gcStore(logp.NewLogger("test"), started, store)

		checkEqualStoreState(t, initState, backend.snapshot())
	})

	t.Run("old state but resource is accessed", func(t *testing.T) {
		const ttl = 60 * time.Second
		started := time.Now().Add(-5 * ttl) // cleanup process is running for a while already

		initState := map[string]state{
			"test::key": {
				TTL:     ttl,
				Updated: started.Add(-ttl),
			},
		}

		backend := createSampleStore(t, initState)
		store := testOpenStore(t, backend)
		defer store.Release()

		// access resource and check it is not gc'ed
		res := store.Get("test::key")
		gcStore(logp.NewLogger("test"), started, store)
		checkEqualStoreState(t, initState, backend.snapshot())

		// release resource and check it gets gc'ed
		res.Release()
		want := map[string]state{}
		gcStore(logp.NewLogger("test"), started, store)
		checkEqualStoreState(t, want, backend.snapshot())
	})

	t.Run("old state but resource has pending updates", func(t *testing.T) {
		const ttl = 60 * time.Second
		started := time.Now().Add(-5 * ttl) // cleanup process is running for a while already

		initState := map[string]state{
			"test::key": {
				TTL:     ttl,
				Updated: started.Add(-ttl),
			},
		}

		backend := createSampleStore(t, initState)
		store := testOpenStore(t, backend)
		defer store.Release()

		// create pending update operation
		res := store.Get("test::key")
		op, err := createUpdateOp(store, res, "test-state-update")
		require.NoError(t, err)
		res.Release()

		// cleanup fails
		gcStore(logp.NewLogger("test"), started, store)
		checkEqualStoreState(t, initState, backend.snapshot())

		// cancel operation (no more pending operations) and try to gc again
		op.done(1)
		gcStore(logp.NewLogger("test"), started, store)
		want := map[string]state{}
		checkEqualStoreState(t, want, backend.snapshot())
	})
}
