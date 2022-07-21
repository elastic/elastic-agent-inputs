// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package memlog

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

var errno0 = syscall.Errno(0)

// syncFile implements the fsync operation for darwin. On darwin fsync is not
// reliable, instead the fcntl syscall with F_FULLFSYNC must be used.
func syncFile(f *os.File) error {
	for {
		_, err := unix.FcntlInt(f.Fd(), unix.F_FULLFSYNC, 0)
		rootCause := errorRootCause(err)
		if err == nil || isIOError(rootCause) {
			return err
		}

		if isRetryErr(err) {
			continue
		}

		err = f.Sync()
		if isRetryErr(err) {
			continue
		}
		return err
	}
}

func isIOError(err error) bool {
	return errors.Is(err, unix.EIO) ||
		// space/quota
		errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT) || errors.Is(err, unix.EFBIG) ||
		// network
		errors.Is(err, unix.ECONNRESET) || errors.Is(err, unix.ENETDOWN) || errors.Is(err, unix.ENETUNREACH)
}

// normalizeSysError returns the underlying error or nil, if the underlying
// error indicates it is no error.
func errorRootCause(err error) error {
	for err != nil {
		u, ok := err.(interface{ Unwrap() error }) //nolint:errorlint // keep old behaviour
		if !ok {
			break
		}
		err = u.Unwrap()
	}

	if err == nil || errors.Is(err, errno0) {
		return nil
	}
	return err
}
