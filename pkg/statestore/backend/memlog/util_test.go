// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package memlog

import (
	"errors"
	"syscall"
	"testing"
)

// A mock Writer implementation that always returns a configurable
// error on the first write call, to test error handling in ensureWriter.
type mockErrorWriter struct {
	errorType     error
	reportedError bool
}

func (mew *mockErrorWriter) Write(data []byte) (n int, err error) {
	if !mew.reportedError {
		mew.reportedError = true
		return 0, mew.errorType
	}
	return len(data), nil
}

func TestEnsureWriter_RetriableError(t *testing.T) {
	// EAGAIN is retriable, ensureWriter.Write should succeed.
	errorWriter := &mockErrorWriter{errorType: syscall.EAGAIN}
	bytes := []byte{1, 2, 3}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write(bytes)
	if err != nil {
		t.Fatalf("ensureWriter shouldn't propagate retriable errors")
	}
	if written != len(bytes) {
		t.Fatalf("Expected %d bytes written, got %d", len(bytes), written)
	}
}

func TestEnsureWriter_NonRetriableError(t *testing.T) {
	// EINVAL is not retriable, ensureWriter.Write should return an error.
	errorWriter := &mockErrorWriter{errorType: syscall.EINVAL}
	bytes := []byte{1, 2, 3}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write(bytes)
	if !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("ensureWriter should propagate nonretriable errors")
	}
	if written != 0 {
		t.Fatalf("Expected 0 bytes written, got %d", written)
	}
}

func TestEnsureWriter_NoError(t *testing.T) {
	// This tests the case where the underlying writer returns with no error,
	// but without writing the full buffer.
	var bytes []byte = []byte{1, 2, 3}
	errorWriter := &mockErrorWriter{errorType: nil}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write(bytes)
	if err != nil {
		t.Fatalf("ensureWriter should only error if the underlying writer does")
	}
	if written != len(bytes) {
		t.Fatalf("Expected %d bytes written, got %d", len(bytes), written)
	}
}
