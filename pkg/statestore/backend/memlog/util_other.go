// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || dragonfly || freebsd || netbsd || openbsd || solaris || aix
// +build linux dragonfly freebsd netbsd openbsd solaris aix

package memlog

import "os"

// syncFile implements the fsync operation for most *nix systems.
// The call is retried if EINTR or EAGAIN is returned.
func syncFile(f *os.File) error {
	// best effort.
	for {
		err := f.Sync()
		if err == nil || !isRetryErr(err) {
			return err
		}
	}
}
