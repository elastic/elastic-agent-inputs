// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shipper

import (
	"time"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type backoffConfig struct {
	Init time.Duration `config:"init" validate:"nonzero"`
	Max  time.Duration `config:"max" validate:"nonzero"`
}

type Config struct {
	Enabled bool `yaml:"enabled"`
	// Server address in the format of host:port, e.g. `localhost:50051`
	Server string `config:"server"`
	// TLS/SSL configurationf or secure connection
	TLS *tlscommon.Config `config:"ssl"`
	// Timeout of a single batch publishing request
	Timeout time.Duration `config:"timeout"`
	// MaxRetries is how many times the same batch is attempted to be sent
	MaxRetries int `config:"max_retries"`
	// BulkMaxSize max amount of events in a single batch
	BulkMaxSize int `config:"bulk_max_size"`
	// Backoff strategy for the shipper output
	Backoff backoffConfig `config:"backoff"`
}

func DefaultConfig() Config {
	return Config{
		Server:      "localhost:50051",
		TLS:         nil,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		BulkMaxSize: 50,
		Backoff: backoffConfig{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
}
