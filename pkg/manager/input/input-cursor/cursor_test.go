// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCursor_IsNew(t *testing.T) {
	t.Run("true if key is not in store", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()

		cursor := makeCursor(store, store.Get("test::key"))
		require.True(t, cursor.IsNew())
	})

	t.Run("true if key is in store but without cursor value", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, map[string]state{
			"test::key": {Cursor: nil},
		}))
		defer store.Release()

		cursor := makeCursor(store, store.Get("test::key"))
		require.True(t, cursor.IsNew())
	})

	t.Run("false if key with cursor value is in persistent store", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, map[string]state{
			"test::key": {Cursor: "test"},
		}))
		defer store.Release()

		cursor := makeCursor(store, store.Get("test::key"))
		require.False(t, cursor.IsNew())
	})

	t.Run("false if key with cursor value is in memory store only", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, map[string]state{
			"test::key": {Cursor: nil},
		}))
		defer store.Release()

		res := store.Get("test::key")
		op, err := createUpdateOp(store, res, "test-state-update")
		require.NoError(t, err)
		defer op.done(1)

		cursor := makeCursor(store, res)
		require.False(t, cursor.IsNew())
	})
}

func TestCursor_Unpack(t *testing.T) {
	t.Run("nothing to unpack if key is new", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, nil))
		defer store.Release()

		var st string
		cursor := makeCursor(store, store.Get("test::key"))

		require.NoError(t, cursor.Unpack(&st))
		require.Equal(t, "", st)
	})

	t.Run("unpack fails if types are not compatible", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, map[string]state{
			"test::key": {Cursor: "test"},
		}))
		defer store.Release()

		var st struct{ A uint }
		cursor := makeCursor(store, store.Get("test::key"))
		require.Error(t, cursor.Unpack(&st))
	})

	t.Run("unpack from state in persistent store", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, map[string]state{
			"test::key": {Cursor: "test"},
		}))
		defer store.Release()

		var st string
		cursor := makeCursor(store, store.Get("test::key"))

		require.NoError(t, cursor.Unpack(&st))
		require.Equal(t, "test", st)
	})

	t.Run("unpack from in memory state if updates are pending", func(t *testing.T) {
		store := testOpenStore(t, createSampleStore(t, map[string]state{
			"test::key": {Cursor: "test"},
		}))
		defer store.Release()

		res := store.Get("test::key")
		op, err := createUpdateOp(store, res, "test-state-update")
		require.NoError(t, err)
		defer op.done(1)

		var st string
		cursor := makeCursor(store, store.Get("test::key"))

		require.NoError(t, cursor.Unpack(&st))
		require.Equal(t, "test-state-update", st)
	})
}
