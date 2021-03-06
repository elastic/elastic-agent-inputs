// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-inputs/pkg/publisher"
	pubtest "github.com/elastic/elastic-agent-inputs/pkg/publisher/testing"
)

func TestPublish(t *testing.T) {
	t.Run("event with cursor state creates update operation", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()
		cursor := makeCursor(store, store.Get("test::key"))

		var actual publisher.Event
		client := &pubtest.FakeClient{
			PublishFunc: func(event publisher.Event) { actual = event },
		}
		p := cursorPublisher{nil, client, &cursor}
		err := p.Publish(publisher.Event{}, "test")
		require.NoError(t, err)

		require.NotNil(t, actual.Private)
	})

	t.Run("event without cursor creates no update operation", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()
		cursor := makeCursor(store, store.Get("test::key"))

		var actual publisher.Event
		client := &pubtest.FakeClient{
			PublishFunc: func(event publisher.Event) { actual = event },
		}
		p := cursorPublisher{nil, client, &cursor}
		err := p.Publish(publisher.Event{}, nil)
		require.NoError(t, err)
		require.Nil(t, actual.Private)
	})

	t.Run("publish returns error if context has been cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()
		cursor := makeCursor(store, store.Get("test::key"))

		p := cursorPublisher{ctx, &pubtest.FakeClient{}, &cursor}
		err := p.Publish(publisher.Event{}, nil)
		require.Equal(t, context.Canceled, err)
	})
}

func TestOp_Execute(t *testing.T) {
	t.Run("applying final op marks the key as finished", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()
		res := store.Get("test::key")

		// create op and release resource. The 'resource' must still be active
		op := mustCreateUpdateOp(t, store, res, "test-updated-cursor-state")
		res.Release()
		require.False(t, res.Finished())

		// this was the last op, the resource should become inactive
		op.Execute(1)
		require.True(t, res.Finished())

		// validate state:
		inSyncCursor := storeInSyncSnapshot(store)["test::key"].Cursor
		inMemCursor := storeMemorySnapshot(store)["test::key"].Cursor
		want := "test-updated-cursor-state"
		assert.Equal(t, want, inSyncCursor)
		assert.Equal(t, want, inMemCursor)
	})

	t.Run("acking multiple ops applies the latest update and marks key as finished", func(t *testing.T) {
		// when acking N events, intermediate updates are dropped in favor of the latest update operation.
		// This test checks that the resource is correctly marked as finished.

		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()
		res := store.Get("test::key")

		// create update operations and release resource. The 'resource' must still be active
		mustCreateUpdateOp(t, store, res, "test-updated-cursor-state-dropped")
		op := mustCreateUpdateOp(t, store, res, "test-updated-cursor-state-final")
		res.Release()
		require.False(t, res.Finished())

		// this was the last op, the resource should become inactive
		op.Execute(2)
		require.True(t, res.Finished())

		// validate state:
		inSyncCursor := storeInSyncSnapshot(store)["test::key"].Cursor
		inMemCursor := storeMemorySnapshot(store)["test::key"].Cursor
		want := "test-updated-cursor-state-final"
		assert.Equal(t, want, inSyncCursor)
		assert.Equal(t, want, inMemCursor)
	})

	t.Run("ACK only subset of pending ops will only update up to ACKed state", func(t *testing.T) {
		// when acking N events, intermediate updates are dropped in favor of the latest update operation.
		// This test checks that the resource is correctly marked as finished.

		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()
		res := store.Get("test::key")

		// create update operations and release resource. The 'resource' must still be active
		op1 := mustCreateUpdateOp(t, store, res, "test-updated-cursor-state-intermediate")
		op2 := mustCreateUpdateOp(t, store, res, "test-updated-cursor-state-final")
		res.Release()
		require.False(t, res.Finished())

		defer op2.done(1) // cleanup after test

		// this was the intermediate op, the resource should still be active
		op1.Execute(1)
		require.False(t, res.Finished())

		// validate state (in memory state is always up to data to most recent update):
		inSyncCursor := storeInSyncSnapshot(store)["test::key"].Cursor
		inMemCursor := storeMemorySnapshot(store)["test::key"].Cursor
		assert.Equal(t, "test-updated-cursor-state-intermediate", inSyncCursor)
		assert.Equal(t, "test-updated-cursor-state-final", inMemCursor)
	})
}

func mustCreateUpdateOp(t *testing.T, store *store, resource *resource, updates interface{}) *updateOp {
	op, err := createUpdateOp(store, resource, updates)
	if err != nil {
		t.Fatalf("Failed to create update op: %v", err)
	}
	return op
}
