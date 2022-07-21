// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storecompliance

import (
	"testing"

	"github.com/elastic/elastic-agent-inputs/pkg/statestore/backend"
)

// Registry helper for writing tests.
// The registry uses a testing.T and provides some MustX methods that fail if
// an error occurred.
type Registry struct {
	T testing.TB
	backend.Registry
}

// Store helper for writing tests.
// The store needs a reference to the Registry with the current test context.
// The Store provides additional helpers for reopening the store, MustX methods
// that will fail the test if an error has occurred.
type Store struct {
	backend.Store

	Registry *Registry
	name     string
}

// Access uses the backend Registry to create a new Store.
func (r *Registry) Access(name string) (*Store, error) {
	s, err := r.Registry.Access(name)
	if err != nil {
		return nil, err
	}
	return &Store{Store: s, Registry: r, name: name}, nil
}

// MustAccess opens a Store. It fails the test if an error has occurred.
func (r *Registry) MustAccess(name string) *Store {
	store, err := r.Access(name)
	must(r.T, err, "open store")
	return store
}

// Close closes the testing store.
func (s *Store) Close() {
	err := s.Store.Close()
	must(s.Registry.T, err, "closing store %q failed", s.name)
}

// ReopenIf reopens the store if b is true.
func (s *Store) ReopenIf(b bool) {
	if b {
		s.Reopen()
	}
}

// Reopen reopens the store by closing the backend store and using the registry
// backend to access the same store again.
func (s *Store) Reopen() {
	t := s.Registry.T

	s.Close()
	if t.Failed() {
		t.Fatal("Test already failed")
	}

	store, err := s.Registry.Registry.Access(s.name)
	must(s.Registry.T, err, "reopen failed")

	s.Store = store
}

// MustHave fails the test if an error occurred in a call to Has.
func (s *Store) MustHave(key string) bool {
	b, err := s.Has(key)
	must(s.Registry.T, err, "unexpected error on store/has call")
	return b
}

// MustGet fails the test if an error occurred in a call to Get.
func (s *Store) MustGet(key string, into interface{}) {
	err := s.Get(key, into)
	must(s.Registry.T, err, "unexpected error on store/get call")
}

// MustSet fails the test if an error occurred in a call to Set.
func (s *Store) MustSet(key string, from interface{}) {
	err := s.Set(key, from)
	must(s.Registry.T, err, "unexpected error on store/set call")
}

// MustRemove fails the test if an error occurred in a call to Remove.
func (s *Store) MustRemove(key string) {
	err := s.Store.Remove(key)
	must(s.Registry.T, err, "unexpected error remove key")
}
