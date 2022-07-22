// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package resources

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// This is the maximum waiting time for goroutine shutdown.
// If the shutdown happens earlier the waiting time will be lower.
// High maximum waiting time was due to flaky tests on CI workers
const defaultFinalizationTimeout = 35 * time.Second

// GoroutinesChecker keeps the count of goroutines when it was created
// so later it can check if this number has increased
type GoroutinesChecker struct {
	before int

	// FinalizationTimeout is the time to wait till goroutines have finished
	FinalizationTimeout time.Duration
}

// NewGoroutinesChecker creates a new GoroutinesChecker
func NewGoroutinesChecker() GoroutinesChecker {
	return GoroutinesChecker{
		before:              runtime.NumGoroutine(),
		FinalizationTimeout: defaultFinalizationTimeout,
	}
}

// Check if the number of goroutines has increased since the checker
// was created
func (c GoroutinesChecker) Check(t testing.TB) {
	t.Helper()
	err := c.check()
	if err != nil {
		dumpGoroutines()
		t.Error(err)
	}
}

func dumpGoroutines() {
	profile := pprof.Lookup("goroutine")
	_ = profile.WriteTo(os.Stdout, 2)
}

func (c GoroutinesChecker) check() error {
	after, err := c.WaitUntilOriginalCount()
	if errors.Is(err, ErrTimeout) {
		return fmt.Errorf("possible goroutines leak, before: %d, after: %d", c.before, after)
	}
	return err
}

// CallAndCheckGoroutines calls a function and checks if it has increased
// the number of goroutines
func CallAndCheckGoroutines(t testing.TB, f func()) {
	t.Helper()
	c := NewGoroutinesChecker()
	f()
	c.Check(t)
}

// ErrTimeout is the error returned when WaitUntilOriginalCount timeouts.
var ErrTimeout = fmt.Errorf("timeout waiting for finalization of goroutines")

// WaitUntilOriginalCount waits until the original number of goroutines are
// present before we created the resource checker.
// It returns the number of goroutines after the check and a timeout error
// in case the wait has expired.
func (c GoroutinesChecker) WaitUntilOriginalCount() (int, error) {
	timeout := time.Now().Add(c.FinalizationTimeout)

	var after int
	for time.Now().Before(timeout) {
		after = runtime.NumGoroutine()
		if after <= c.before {
			return after, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return after, ErrTimeout
}

// WaitUntilIncreased waits till the number of goroutines is n plus the number
// before creating the checker.
func (c *GoroutinesChecker) WaitUntilIncreased(n int) {
	for runtime.NumGoroutine() < c.before+n {
		time.Sleep(10 * time.Millisecond)
	}
}
