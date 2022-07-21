// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package memlog

import "os"

// syncFile implements the fsync operation for Windows. Internally
// FlushFileBuffers will be used.
func syncFile(f *os.File) error {
	return f.Sync() // stdlib already uses FlushFileBuffers, yay
}
