// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package memlog

import "errors"

var (
	errRegClosed   = errors.New("registry has been closed")
	errLogInvalid  = errors.New("can not add operation to log file, a checkpoint is required")
	errTxIDInvalid = errors.New("invalid update sequence number")
	errKeyUnknown  = errors.New("key unknown")
)
